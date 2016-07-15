package sms

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/zap"
	"testing"
)

func getTestReq() *SMSReq {
	return &SMSReq{
		Category:     "test",
		TemplateID:   "0000000",
		PhoneNumbers:[]string{"1000000", "1000001", "1000002"},
	}
}

func TestSend_Filter(t *testing.T) {
	req := getTestReq()
	RegisterFilter(req.Category, func(ctx *Context, req *SMSReq, resp *SMSResp) (exit bool) {
		resp.Fail = append(resp.Fail, FailReq{
			PhoneNumber: req.PhoneNumbers[0],
			FailReason:  "i don't know",
		})
		req.PhoneNumbers = req.PhoneNumbers[1:]
		return
	})
	defer ResetFilters(req.Category, nil)

	selector := &RandomSelector{}
	selector.AddSender(req.Category, &MockSender{})

	logger := zap.NewJSON()
	//logger.SetLevel(zap.DebugLevel)
	logger.SetLevel(zap.InfoLevel)

	ctx := &Context{
		Selector: selector,
		Logger:   logger,
	}

	resp := Send(ctx, req)
	require.NotEmpty(t, resp, "should not return empty response")
	require.NotEmpty(t, resp.Fail, "should have fail message")
	require.Equal(t, 1, len(resp.Fail), "should one fail message")
	require.Equal(t, "1000000", resp.Fail[0].PhoneNumber, "first phone number failed")
}

func TestSend_Sender(t *testing.T) {
	req := getTestReq()
	selector := &RandomSelector{}
	selector.AddSender(req.Category, SenderFunc(func(ctx *Context, req2 *SMSReq, resp *SMSResp) {
		assert.Equal(t, req, req2, "check request")
		require.NotEmpty(t, resp, "response shouldn't be nil")
		resp.Code = CodeSuccess
	}))

	resp := Send(&Context{
		Selector: selector,
	}, req)
	require.NotEmpty(t, resp)
	assert.Equal(t, CodeSuccess, resp.Code)
	assert.Empty(t, resp.Fail)
}
