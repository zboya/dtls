package dtls

import (
	"errors"
)

type handshakeFragmentList struct {
	MsgType    handshakeType
	Length     uint32
	MessageSeq uint16
	Fragments  []*handshake
}

func newHandshakeFragmentList(h *handshake) *handshakeFragmentList {
	return &handshakeFragmentList{
		MsgType:    h.MsgType,
		Length:     h.Length,
		MessageSeq: h.MessageSeq,
		Fragments:  []*handshake{h},
	}
}

func (hfl *handshakeFragmentList) InsertFragment(newHandshake *handshake) error {
	if newHandshake.MsgType != hfl.MsgType ||
		newHandshake.Length != hfl.Length ||
		newHandshake.MessageSeq != hfl.MessageSeq {
		return errors.New("Received a handshake fragment which is incompatible with previous fragments")
	}
	for i, handshake := range hfl.Fragments {
		if handshake.FragmentOffset > newHandshake.FragmentOffset {
			hfl.InsertFragmentAt(newHandshake, i)
			return nil
		}
	}
	hfl.InsertFragmentAt(newHandshake, len(hfl.Fragments))
	return nil
}

func (hfl *handshakeFragmentList) InsertFragmentAt(f *handshake, i int) {
	hfl.Fragments = append(hfl.Fragments, nil)
	copy(hfl.Fragments[i+1:], hfl.Fragments[i:])
	hfl.Fragments[i] = f
}

func (hfl *handshakeFragmentList) IsComplete() bool {
	if len(hfl.Fragments) == 1 &&
		hfl.Fragments[0].FragmentOffset == 0 &&
		hfl.Fragments[0].FragmentLength == hfl.Fragments[0].Length {
		return true
	}
	offset := uint32(0)
	for _, handshake := range hfl.Fragments {
		if handshake.FragmentOffset <= offset {
			offset = handshake.FragmentOffset + handshake.FragmentLength
		} else {
			return false
		}
	}
	if offset == hfl.Length {
		return true
	}
	return false
}

func (hfl *handshakeFragmentList) GetCompleteHandshake() *handshake {
	if len(hfl.Fragments) == 1 &&
		hfl.Fragments[0].FragmentOffset == 0 &&
		hfl.Fragments[0].FragmentLength == hfl.Fragments[0].Length {
		return hfl.Fragments[0]
	}
	h := &handshake{
		MsgType:        hfl.MsgType,
		Length:         hfl.Length,
		MessageSeq:     hfl.MessageSeq,
		FragmentOffset: 0,
		FragmentLength: hfl.Length,
		Fragment:       make([]byte, hfl.Length),
	}
	for _, handshake := range hfl.Fragments {
		copy(h.Fragment[handshake.FragmentOffset:], handshake.Fragment)
	}
	return h
}
