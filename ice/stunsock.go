package ice

import (
	"net"
	"sync"
	"time"

	"fmt"

	"github.com/nkbai/log"
	"github.com/nkbai/goice/stun"
)

const defaultReadDeadLine = time.Second * 10

/*
用于在没有 turn server 而只有 stun server 的情形下,收集本机候选地址.
*/
type stunSocket struct {
	ServerAddr   string
	MappedAddr   net.UDPAddr
	LocalAddr    string // local addr used to  connect server
	Client       *stun.Client
	ReadDeadline time.Duration
	localAddrs   []string //for listen
}

func newStunSocket(serverAddr string) (s *stunSocket, err error) {
	s = &stunSocket{
		ServerAddr:   serverAddr,
		ReadDeadline: defaultReadDeadLine,
	}
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		log.Crit(fmt.Sprintf("failed to dial:%s", err))
	}
	client, err := stun.NewClient(stun.ClientOptions{
		Connection: conn,
	})
	if err != nil {
		return
	}
	s.Client = client
	s.LocalAddr = conn.(*net.UDPConn).LocalAddr().String()
	return
}

//get mapped address from server
func (s *stunSocket) mapAddress() error {
	deadline := time.Now().Add(s.ReadDeadline)
	var err error
	wg := sync.WaitGroup{}
	wg.Add(1)
	err = s.Client.Do(stun.MustBuild(stun.TransactionIDSetter, stun.BindingRequest), deadline, func(res stun.Event) {
		defer wg.Done()
		if res.Error != nil {
			err = res.Error
			return
		}
		var xorAddr stun.XORMappedAddress
		if err = xorAddr.GetFrom(res.Message); err != nil {
			var addr stun.MappedAddress
			err = addr.GetFrom(res.Message)
			if err != nil {
				return
			}
			s.MappedAddr = net.UDPAddr{IP: addr.IP, Port: addr.Port}
		} else {
			s.MappedAddr = net.UDPAddr{IP: xorAddr.IP, Port: xorAddr.Port}
		}
	})
	wg.Wait()
	//keep alive todo
	return err
}

/*
获取有一部分信息的candidiate.第一个是本机主要地址,最后一个是缺省 Candidate
*/
func (s *stunSocket) GetCandidates() (candidates []*Candidate, err error) {
	err = s.mapAddress()
	if err != nil {
		return
	}
	c := new(Candidate)
	c.baseAddr = s.LocalAddr
	c.Type = CandidateServerReflexive
	c.addr = s.MappedAddr.String()
	c.Foundation = calcFoundation(c.baseAddr)
	candidates, err = getLocalCandidates(c.baseAddr)
	if err != nil {
		return
	}
	for _, c := range candidates {
		s.localAddrs = append(s.localAddrs, c.addr)
	}
	if c.addr != c.baseAddr { //we have a public ip
		candidates = append(candidates, c)
	}
	return
}

func (s *stunSocket) Close() {
	if s.Client != nil {
		s.Client.Close()
	}
}

/*
address need to listen for input stun binding request...
*/
func (s *stunSocket) getListenCandidiates() []string {
	return s.localAddrs
}
