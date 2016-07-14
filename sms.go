package sms

import (
	"fmt"
	"github.com/uber-go/zap"
)

type SMSReq struct {
	Category     string
	PhoneNumbers []string
	TemplateID   string
	Values       [][]string
}

type SMSResp struct {
	ID      string
	Code    int
	Message string
	Fail    []FailReq
}

type FailReq struct {
	PhoneNumber string
	FailReason  string
}

type Context struct {
	Logger   zap.Logger
	Selector Selector
	IDGen    IDGen
}

func Send(ctx *Context, req *SMSReq) (resp *SMSResp) {
	if ctx.Logger == nil {
		ctx.Logger = zap.NewJSON()
	}
	if ctx.Selector == nil {
		ctx.Selector = &RandomSelector{}
	}
	if ctx.IDGen == nil {
		idGen, err := NewDefaultIDGen()
		if err != nil {
			ctx.Logger.Error("cann't new default id generator", zap.Error(err))
		} else {
			ctx.IDGen = idGen
		}
	}

	resp = &SMSResp{
		ID: ctx.IDGen.Next(),
	}

	if pnl, vl := len(req.PhoneNumbers), len(req.Values); pnl != vl {
		resp.Code = CodeInvalidParam
		resp.Message = fmt.Sprintf("phoneNumbers.length(%d) != values.length(%d)", pnl, vl)
		return
	}

	filtersRWM.RLock()
	exit := false
	for _, f := range filters[req.Category] {
		if exit = f(ctx, req, resp); exit {
			break
		}
	}
	filtersRWM.RUnlock()
	if exit || len(req.PhoneNumbers) == 0 {
		return
	}

	sender, errCode, err := ctx.Selector.Select(req.Category)
	if err != nil {
		resp.Code = errCode
		resp.Message = err.Error()
		return
	}
	sender.Send(ctx, req, resp)

	return
}
