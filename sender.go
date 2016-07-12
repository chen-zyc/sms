package sms

import (
	"github.com/uber-go/zap"
	"strings"
)

type Sender interface {
	Send(ctx *Context, req *SMSReq, resp *SMSResp)
}

type SenderFunc func(ctx *Context, req *SMSReq, resp *SMSResp)

func (sf SenderFunc) Send(ctx *Context, req *SMSReq, resp *SMSResp) {
	sf(ctx, req, resp)
}

type mockSender struct{}

func (ms *mockSender) Send(ctx *Context, req *SMSReq, resp *SMSResp) {
	logger := ctx.Logger.With(
		zap.String("caller", caller()),
		zap.String("category", req.Category),
		zap.String("template", req.TemplateID),
	)

	for i, pn := range req.PhoneNumbers {
		logger.Debug("send sms",
			zap.String("phone", pn),
			zap.String("values", strings.Join(req.Values[i], ",")),
		)
	}
	resp.Code = CodeSuccess
}
