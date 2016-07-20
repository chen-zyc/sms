package sms

import (
	"github.com/uber-go/zap"
)

type SMSReq struct {
	Category     string
	TemplateID   string
	PhoneNumbers []string
	Args         []string
	Content      string
}

type SMSResp struct {
	ID      string
	Code    int32
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

	filtersRWM.RLock()
	exit := false
	for _, f := range filters[FilterGlobal] {
		if exit = f(ctx, req, resp); exit {
			break
		}
	}
	if !exit {
		for _, f := range filters[req.Category] {
			if exit = f(ctx, req, resp); exit {
				break
			}
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
