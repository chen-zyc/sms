package sms

import "sync"

var filters = make(map[string][]Filter)
var filtersRWM sync.RWMutex

// Filter 发送之前调用该方法。
// 如果需要去掉某些不合法的手机号，必须从req.PhoneNumber和req.Values中将对应的内容删除，并在resp.Fail中添加删除的原因
type Filter func(ctx *Context, req *SMSReq, resp *SMSResp) (exit bool)

func RegisterFilter(category string, f Filter) {
	if f == nil {
		return
	}
	filtersRWM.Lock()
	if fs, ok := filters[category]; ok {
		fs = append(fs, f)
		filters[category] = fs
	} else {
		filters[category] = []Filter{f}
	}
	filtersRWM.Unlock()
}

func ResetFilters(category string, newFilters []Filter) {
	filtersRWM.Lock()
	filters[category] = newFilters
	filtersRWM.Unlock()
}
