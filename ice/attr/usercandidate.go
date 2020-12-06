package attr

import "github.com/nkbai/goice/stun"

type useCandidateSetter struct{}

func (useCandidateSetter) AddTo(m *stun.Message) error {
	m.Add(stun.AttrUseCandidate, nil)
	return nil
}

// UseCandidate is Setter for m.UseCandidate.
var UseCandidate stun.Setter = useCandidateSetter{}
