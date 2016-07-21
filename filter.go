package sms

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	FilterGlobal = "_GLOBAL"
)

var filters = make(map[string][]Filter)
var filtersRWM sync.RWMutex

var (
	ErrExceedLimit = errors.New("exceed limit")
	ErrTryAgain    = errors.New("try again")
)

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

// RateLimitFilterRedis 基于redis对手机号做发送限制
type RateLimitFilterRedis struct {
	RedisPool    *redis.Pool
	MaxTokens    int64
	Tokens       int64
	PerSec       int64
	MaxTryTimes  int
	KeyExpireSec int                             // 键过期秒数
	KeyFunc      func(req *SMSReq, i int) string // 键生成函数
}

func NewRateLimitFilterRedis(redisPool *redis.Pool, maxTokens, tokens int64, per time.Duration) *RateLimitFilterRedis {
	return &RateLimitFilterRedis{
		RedisPool:    redisPool,
		MaxTokens:    maxTokens,
		Tokens:       tokens,
		PerSec:       int64(per.Seconds()),
		MaxTryTimes:  5,
		KeyExpireSec: 60,
	}
}

func (rl *RateLimitFilterRedis) FilterFunc() Filter {
	return func(ctx *Context, req *SMSReq, resp *SMSResp) (exit bool) {
		pns, failed := rl.Filter(ctx, req)
		req.PhoneNumbers = pns
		resp.Fail = append(resp.Fail, failed...)
		return len(pns) == 0
	}
}

func (rl *RateLimitFilterRedis) Filter(ctx *Context, req *SMSReq) ([]string, []FailReq) {
	if rl.RedisPool == nil {
		return req.PhoneNumbers, nil
	}
	c := rl.RedisPool.Get()
	defer c.Close()

	var (
		newNumbers = make([]string, 0, len(req.PhoneNumbers))
		failed     []FailReq
	)

LOOP:
	for i := 0; i < len(req.PhoneNumbers); i++ {
		key := req.PhoneNumbers[i]
		if rl.KeyFunc != nil {
			key = rl.KeyFunc(req, i)
		}
		var err error
		for j := 0; j < rl.MaxTryTimes; j++ {
			err = rl.checkLimit(ctx, key, c)
			if err == nil {
				newNumbers = append(newNumbers, req.PhoneNumbers[i])
				continue LOOP
			}
			if err == ErrExceedLimit {
				failed = append(failed, FailReq{
					PhoneNumber: req.PhoneNumbers[i],
					FailReason:  err.Error(),
				})
				continue LOOP
			}
			if err == ErrTryAgain {
				continue
			}
		}
		// 超过最大次数还是有错误
		if err != nil {
			failed = append(failed, FailReq{
				PhoneNumber: req.PhoneNumbers[i],
				FailReason:  err.Error(),
			})
		} else {
			failed = append(failed, FailReq{
				PhoneNumber: req.PhoneNumbers[i],
				FailReason:  "cann't acquire access,max try times:" + strconv.Itoa(rl.MaxTryTimes),
			})
		}
	}
	return newNumbers, failed
}

// 这种实现在高并发下也可以做到准确限速，缺点是执行的redis命令多，性能略低
func (rl *RateLimitFilterRedis) checkLimit(ctx *Context, key string, c redis.Conn) error {
	reply, err := redis.String(c.Do("WATCH", key))
	if err != nil {
		return err
	}
	if reply != "OK" {
		return errors.New("exec WATCH reply not OK")
	}

	reply, err = redis.String(c.Do("GET", key))
	if err != nil && err != redis.ErrNil {
		return err
	} else {
		err = nil
	}

	var nowSec = time.Now().Unix()
	var lastSec int64
	var tokens int64
	replyParts := strings.SplitN(reply, ",", 2)
	if len(replyParts) == 2 {
		lastSec, err = strconv.ParseInt(replyParts[0], 10, 64)
		if err != nil {
			lastSec = nowSec
			err = nil
		}
		tokens, err = strconv.ParseInt(replyParts[1], 10, 64)
		if err != nil {
			tokens = rl.MaxTokens
			err = nil
		}
	} else {
		lastSec = nowSec
		tokens = rl.MaxTokens
	}

	// sync
	diff := nowSec - lastSec
	tokensToPut := rl.Tokens * diff / rl.PerSec
	if tokensToPut > 0 {
		tokens += tokensToPut
		if tokens > rl.MaxTokens {
			tokens = rl.MaxTokens
		}
		lastSec = nowSec
	}

	if tokens < 1 {
		return ErrExceedLimit
	}
	tokens--

	reply, err = redis.String(c.Do("MULTI"))
	if err != nil {
		return err
	}
	if reply != "OK" {
		return errors.New("exec MULTI reply not OK")
	}

	c.Do("SET", key, fmt.Sprintf("%d,%d", lastSec, tokens))
	c.Do("EXPIRE", key, rl.KeyExpireSec)
	replyIntf, err := c.Do("EXEC")

	if err != nil {
		if err == redis.ErrNil {
			return ErrTryAgain
		}
		return err
	}

	if replyIntf == nil {
		return ErrTryAgain
	}

	return nil
}

type RateLimitFilterRedisCounter struct {
	RedisPool    *redis.Pool
	Count        int
	KeyFunc      func(req *SMSReq, i int) string
	KeyExpireSec int
}

func NewRateLimitFilterRedisCounter(redisPool *redis.Pool, count int, keyExpireSec int) *RateLimitFilterRedisCounter {
	return &RateLimitFilterRedisCounter{
		RedisPool:    redisPool,
		Count:        count,
		KeyExpireSec: keyExpireSec,
	}
}

func (c *RateLimitFilterRedisCounter) FilterFunc() Filter {
	return func(ctx *Context, req *SMSReq, resp *SMSResp) (exit bool) {
		pns, failed := c.Filter(ctx, req)
		req.PhoneNumbers = pns
		resp.Fail = append(resp.Fail, failed...)
		return len(pns) == 0
	}
}

func (c *RateLimitFilterRedisCounter) Filter(ctx *Context, req *SMSReq) ([]string, []FailReq) {
	if c.RedisPool == nil {
		return req.PhoneNumbers, nil
	}
	conn := c.RedisPool.Get()
	defer conn.Close()

	var (
		newNumbers = make([]string, 0, len(req.PhoneNumbers))
		failed     []FailReq
	)

	for i := 0; i < len(req.PhoneNumbers); i++ {
		key := req.PhoneNumbers[i]
		if c.KeyFunc != nil {
			key = c.KeyFunc(req, i)
		}
		err := c.checkLimit(ctx, conn, key)
		if err != nil {
			failed = append(failed, FailReq{
				PhoneNumber: req.PhoneNumbers[i],
				FailReason:  err.Error(),
			})
		} else {
			newNumbers = append(newNumbers, req.PhoneNumbers[i])
		}
	}
	return newNumbers, failed
}

// 这种实现在高并发下不能准确的限速，性能比RateLimitFilterRedis要好大约一倍
func (c *RateLimitFilterRedisCounter) checkLimit(ctx *Context, conn redis.Conn, key string) error {
	count, err := redis.Int(conn.Do("GET", key))

	if err == nil && count >= c.Count {
		return ErrExceedLimit
	}

	if err == redis.ErrNil { // 没有这个key
		_, err = conn.Do("MULTI")
		if err != nil {
			return err
		}
		conn.Do("INCR", key)
		conn.Do("EXPIRE", key, c.KeyExpireSec)
		_, err = conn.Do("EXEC")
		if err != nil {
			return err
		}
	} else {
		_, err = conn.Do("INCR", key)
		if err != nil {
			return err
		}
	}
	return nil
}

type ContentFilter struct{}

func (cf *ContentFilter) FilterFunc() Filter {
	return func(ctx *Context, req *SMSReq, resp *SMSResp) (exit bool) {
		content, err := cf.Filter(req.TemplateID, req.Args)
		if err != nil {
			resp.Code = CodeInvalidParam
			resp.Message = err.Error()
			return false
		}
		req.Content = content
		return true
	}
}

func (cf *ContentFilter) Filter(templateID string, args []string) (content string, err error) {
	temp := FindTemplate(templateID)
	if temp == nil {
		return "", errors.New("cann't find template:" + templateID)
	}

	content, err = temp.SMSContent(args)
	return
}
