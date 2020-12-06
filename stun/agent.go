package stun

import (
	"errors"
	"sync"
	"time"
)

// AgentOptions are required to initialize Agent.
type AgentOptions struct {
	Handler Handler // Default handler, can be nil.
}

// NewAgent initializes and returns new Agent from options.
func NewAgent(o AgentOptions) *Agent {
	a := &Agent{
		transactions: make(map[TransactionID]agentTransaction),
		zeroHandler:  o.Handler,
	}
	return a
}

// Agent is low-level abstraction over transaction list that
// handles concurrency (all calls are goroutine-safe) and
// time outs (via Collect call).
type Agent struct {
	// transactions is map of transactions that are currently
	// in progress. Event handling is done in such way when
	// transaction is unregistered before agentTransaction access,
	// minimizing mux lock and protecting agentTransaction from
	// data races via unexpected concurrent access.
	transactions map[TransactionID]agentTransaction
	closed       bool       // all calls are invalid if true
	mux          sync.Mutex // protects transactions and closed
	zeroHandler  Handler    // handles non-registered transactions if set
}

// Handler handles state changes of transaction.
type Handler interface {
	// HandleEvent is called on transaction state change.
	// Usage of e is valid only during call, user must
	// copy needed fields explicitly.
	HandleEvent(e Event)
}

// HandlerFunc is function that implements Handler interface.
type HandlerFunc func(e Event)

// HandleEvent implements Handler.
func (f HandlerFunc) HandleEvent(e Event) {
	f(e)
}

// Event is set of arguments passed to AgentFn, describing
// an transaction event. Do not reuse outside AgentFn.
type Event struct {
	Message *Message
	Error   error
}

// agentTransaction represents transaction in progress.
// If transaction is succeed or failed, f will be called
// provided by event.
// Concurrent access is invalid.
type agentTransaction struct {
	id       TransactionID
	deadline time.Time
	h        Handler
}

var (
	// ErrTransactionStopped indicates that transaction was manually stopped.
	ErrTransactionStopped = errors.New("transaction is stopped")
	// ErrTransactionNotExists indicates that agent failed to find transaction.
	ErrTransactionNotExists = errors.New("transaction not exists")
	// ErrTransactionExists indicates that transaction with same id is already
	// registered.
	ErrTransactionExists = errors.New("transaction exists with same id")
)

// StopWithError removes transaction from list and calls transaction callback
// with provided error. Can return ErrTransactionNotExists and ErrAgentClosed.
func (a *Agent) StopWithError(id TransactionID, err error) error {
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}
	t, exists := a.transactions[id]
	delete(a.transactions, id)
	a.mux.Unlock()
	if !exists {
		return ErrTransactionNotExists
	}
	t.h.HandleEvent(Event{
		Error: err,
	})
	return nil
}

// Stop stops transaction by id with ErrTransactionStopped, blocking
// until callback returns.
func (a *Agent) Stop(id TransactionID) error {
	return a.StopWithError(id, ErrTransactionStopped)
}

// ErrAgentClosed indicates that agent is in closed state and is unable
// to handle transactions.
var ErrAgentClosed = errors.New("agent is closed")

// Start registers transaction with provided id, deadline and callback.
// Could return ErrAgentClosed, ErrTransactionExists.
// Callback f is guaranteed to be eventually called. See AgentFn for
// callback processing constraints.
func (a *Agent) Start(id TransactionID, deadline time.Time, h Handler) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	if a.closed {
		return ErrAgentClosed
	}
	_, exists := a.transactions[id]
	if exists {
		return ErrTransactionExists
	}
	a.transactions[id] = agentTransaction{
		id:       id,
		h:        h,
		deadline: deadline,
	}
	return nil
}

// agentCollectCap is initial capacity for Agent.Collect slices,
// sufficient to make function zero-alloc in most cases.
const agentCollectCap = 100

// ErrTransactionTimeOut indicates that transaction has reached deadline.
var ErrTransactionTimeOut = errors.New("transaction is timed out")

// Collect terminates all transactions that have deadline before provided
// time, blocking until all handlers will process ErrTransactionTimeOut.
// Will return ErrAgentClosed if agent is already closed.
//
// It is safe to call Collect concurrently but makes no sense.
func (a *Agent) Collect(gcTime time.Time) error {
	toCall := make([]Handler, 0, agentCollectCap)
	toRemove := make([]TransactionID, 0, agentCollectCap)
	a.mux.Lock()
	if a.closed {
		// Doing nothing if agent is closed.
		// All transactions should be already closed
		// during Close() call.
		a.mux.Unlock()
		return ErrAgentClosed
	}
	// Adding all transactions with deadline before gcTime
	// to toCall and toRemove slices.
	// No allocs if there are less than agentCollectCap
	// timed out transactions.
	for id, t := range a.transactions {
		if t.deadline.Before(gcTime) {
			toRemove = append(toRemove, id)
			toCall = append(toCall, t.h)
		}
	}
	// Un-registering timed out transactions.
	for _, id := range toRemove {
		delete(a.transactions, id)
	}
	// Calling callbacks does not require locked mutex,
	// reducing lock time.
	a.mux.Unlock()
	// Sending ErrTransactionTimeOut to all callbacks, blocking
	// Collect until last one.
	event := Event{
		Error: ErrTransactionTimeOut,
	}
	for _, handler := range toCall {
		handler.HandleEvent(event)
	}
	return nil
}

// Process incoming message, picking handler by transaction id.
// If transaction is not registered, zero handler is used. If default
// handle is not provided, message is silently ignored.
// Call blocks until handler returns.
func (a *Agent) Process(m *Message) error {
	e := Event{
		Message: m,
	}
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}
	t, ok := a.transactions[m.TransactionID]
	delete(a.transactions, m.TransactionID)
	a.mux.Unlock()
	if ok {
		t.h.HandleEvent(e)
	} else if a.zeroHandler != nil {
		a.zeroHandler.HandleEvent(e)
	}
	return nil
}

// Close terminates all transactions with ErrAgentClosed and renders Agent to
// closed state.
func (a *Agent) Close() error {
	e := Event{
		Error: ErrAgentClosed,
	}
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}
	for _, t := range a.transactions {
		t.h.HandleEvent(e)
	}
	a.transactions = nil
	a.closed = true
	a.zeroHandler = nil
	a.mux.Unlock()
	return nil
}
