package sms

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/zap"
	"strings"
	"testing"
)

func getTestReq() *SMSReq {
	return &SMSReq{
		Category:     "test",
		TemplateID:   "0000000",
		PhoneNumbers: []string{"1000000", "1000001", "1000002"},
		Values: [][]string{
			strings.Split("v1,v2,v3", ","),
			strings.Split("v4,v5,v6", ","),
			strings.Split("v7,v8,v9", ","),
		},
	}
}

func TestSend_InvalidParam(t *testing.T) {
	req := getTestReq()
	req.Values = req.Values[:len(req.Values)-1] // 少一个参数

	resp := Send(&Context{}, req)
	require.NotEmpty(t, resp, "should not return empty response")
	assert.Equal(t, CodeInvalidParam, resp.Code, "invalid param error code")
	assert.Equal(t,
		fmt.Sprintf("phoneNumbers.length(%d) != values.length(%d)", len(req.PhoneNumbers), len(req.Values)),
		resp.Message,
		"error message",
	)
	assert.Empty(t, resp.Fail, "no fail list")
}

func TestSend_Filter(t *testing.T) {
	req := getTestReq()
	RegisterFilter(req.Category, func(ctx *Context, req *SMSReq, resp *SMSResp) (exit bool) {
		failPhone := req.PhoneNumbers[0]
		req.PhoneNumbers = req.PhoneNumbers[1:]
		req.Values = req.Values[1:]
		resp.Fail = append(resp.Fail, FailReq{
			PhoneNumber: failPhone,
			FailReason:  "i don't know",
		})
		return
	})
	defer ResetFilters(req.Category, nil)

	selector := &RandomSelector{}
	selector.AddSender(req.Category, &mockSender{})

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
