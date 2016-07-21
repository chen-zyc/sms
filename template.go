package sms

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
)

var templates = map[string]*SMSTemplate{}
var templatesRWM sync.RWMutex

func RegisterTemplate(id string, t *SMSTemplate) {
	templatesRWM.Lock()
	_, exist := templates[id]
	if exist && t == nil { // 删除
		delete(templates, id)
	} else if t != nil { // 替换或新增
		templates[id] = t
	}
	templatesRWM.Unlock()
}

func FindTemplate(id string) *SMSTemplate {
	templatesRWM.RLock()
	t := templates[id]
	templatesRWM.RUnlock()
	return t
}

type SMSTemplate struct {
	TempID        string
	Temp          string
	NumArgs       int
	valueCheckers []ValueChecker
}

func (t SMSTemplate) SMSContent(args []string) (content string, err error) {
	if t.NumArgs != len(args) {
		err = fmt.Errorf("template need %d args, provide %d", t.NumArgs, len(args))
		return
	}

	for i, v := range args {
		c := t.valueCheckers[i]
		if c == nil {
			continue
		}
		err = c.Check(v)
		if err != nil {
			return
		}
	}

	argTemp := make([]interface{}, len(args))
	for i, a := range args {
		argTemp[i] = a
	}
	content = fmt.Sprintf(t.Temp, argTemp...)
	return
}

type ValueChecker interface {
	Check(v string) error
}

type RegexpChecker struct {
	reg *regexp.Regexp
}

func NewRegexpChecker(expr string) (*RegexpChecker, error) {
	reg, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}
	return &RegexpChecker{
		reg: reg,
	}, nil
}

func (rc *RegexpChecker) Check(v string) error {
	matched := rc.reg.MatchString(v)
	if !matched {
		return errors.New("invalid arg:" + v)
	}
	return nil
}

type IntChecker struct{}

func (ic *IntChecker) Check(v string) error {
	_, err := strconv.ParseInt(v, 10, 64)
	return err
}
