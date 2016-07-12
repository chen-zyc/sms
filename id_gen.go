package sms

import (
	"github.com/sdming/gosnow"
	"strconv"
)

type IDGen interface {
	Next() string
}

type DefaultIDGen struct {
	idGen *gosnow.SnowFlake
}

func NewDefaultIDGen() (*DefaultIDGen, error) {
	idGen, err := gosnow.Default()
	if err != nil {
		return nil, err
	}
	return &DefaultIDGen{
		idGen: idGen,
	}, nil
}

func (g DefaultIDGen) Next() string {
	if g.idGen == nil {
		return ""
	}
	id, err := g.idGen.Next()
	if err != nil {
		return ""
	}
	return strconv.FormatUint(id, 10)
}
