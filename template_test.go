package sms

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func buildTestTemplate() (*SMSTemplate, error) {
	c1, err := NewRegexpChecker("^[1-9]{4,6}$")
	if err != nil {
		return nil, err
	}
	c2, err := NewRegexpChecker("^.{3}$")
	if err != nil {
		return nil, err
	}

	temp := &SMSTemplate{
		TempID:        "001",
		Temp:          "[%s] XX验证码，30分钟内有效【%s】",
		NumArgs:       2,
		valueCheckers: []ValueChecker{c1, c2},
	}
	return temp, nil
}

func TestSMSTemplate(t *testing.T) {
	temp, err := buildTestTemplate()
	if err != nil {
		t.Fatal(err)
	}

	requests := []struct {
		args    []string
		content string
		err     string
	}{
		{[]string{"1234", "abc"}, "[1234] XX验证码，30分钟内有效【abc】", ""},
		{[]string{"123456", "abc"}, "[123456] XX验证码，30分钟内有效【abc】", ""},
		{[]string{"123", "abcd"}, "", "invalid arg:123"},
		{[]string{"1234", "abcde"}, "", "invalid arg:abcde"},
		{[]string{"1234"}, "", "template need 2 args, provide 1"},
		{[]string{"0123", "abcd"}, "", "invalid arg:0123"},
	}

	for i, req := range requests {
		msg := fmt.Sprintf("#%d: %v", i, req.args)
		content, err := temp.SMSContent(req.args)
		if req.err == "" {
			require.Empty(t, err, msg)
		} else {
			require.NotEmpty(t, err, msg, content)
			assert.Equal(t, req.err, err.Error(), msg, content)
		}
		assert.Equal(t, req.content, content, msg)
	}
}
