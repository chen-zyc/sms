package sms

import (
	"errors"
	"math/rand"
	"sync"
)

type Selector interface {
	Select(category string) (sender Sender, errCode int, err error)
}

type RandomSelector struct {
	senders map[string][]Sender
	sync.RWMutex
}

func (rs *RandomSelector) Select(category string) (sender Sender, errCode int, err error) {
	rs.RLock()
	if senders, ok := rs.senders[category]; ok && len(senders) > 0 {
		sender = senders[rand.Intn(len(senders))]
	} else {
		errCode = CodeNoSender
		err = errors.New("no sender under " + category)
	}
	rs.RUnlock()
	return
}

func (rs *RandomSelector) AddSender(category string, s Sender) {
	if rs.senders == nil {
		rs.senders = make(map[string][]Sender)
	}
	if senders, ok := rs.senders[category]; ok {
		senders = append(senders, s)
		rs.senders[category] = senders
	} else {
		rs.senders[category] = []Sender{s}
	}
}
