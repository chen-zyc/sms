package sms

import (
	"github.com/uber-go/zap"
)

type Sender interface {
	Send(ctx *Context, req *SMSReq, resp *SMSResp)
}

type SenderFunc func(ctx *Context, req *SMSReq, resp *SMSResp)

func (sf SenderFunc) Send(ctx *Context, req *SMSReq, resp *SMSResp) {
	sf(ctx, req, resp)
}

type MockSender struct{}

func (ms *MockSender) Send(ctx *Context, req *SMSReq, resp *SMSResp) {
	ctx.Logger.Debug(
		"send sms",
		zap.String("caller", caller()),
		zap.String("category", req.Category),
		zap.String("template", req.TemplateID),
		zap.Object("args", req.Args),
		zap.Object("phoneNumbers", req.PhoneNumbers),
	)
	resp.Code = CodeSuccess
}
