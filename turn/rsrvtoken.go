package turn

import "github.com/nkbai/goice/stun"

// ReservationToken represents RESERVATION-TOKEN attribute.
//
// The RESERVATION-TOKEN attribute contains a token that uniquely
// identifies a relayed transport address being held in reserve by the
// server. The server includes this attribute in a success response to
// tell the client about the token, and the client includes this
// attribute in a subsequent Allocate request to request the server use
// that relayed transport address for the allocation.
//
// https://trac.tools.ietf.org/html/rfc5766#section-14.9
type ReservationToken []byte

const reservationTokenSize = 8 // 8 bytes

// AddTo adds RESERVATION-TOKEN to message.
func (t ReservationToken) AddTo(m *stun.Message) error {
	if len(t) != reservationTokenSize {
		return &BadAttrLength{
			Attr:     stun.AttrReservationToken,
			Expected: reservationTokenSize,
			Got:      len(t),
		}
	}
	m.Add(stun.AttrReservationToken, t)
	return nil
}

// GetFrom decodes RESERVATION-TOKEN from message.
func (t *ReservationToken) GetFrom(m *stun.Message) error {
	v, err := m.Get(stun.AttrReservationToken)
	if err != nil {
		return err
	}
	if len(v) != reservationTokenSize {
		return &BadAttrLength{
			Attr:     stun.AttrReservationToken,
			Expected: reservationTokenSize,
			Got:      len(v),
		}
	}
	*t = v
	return nil
}
