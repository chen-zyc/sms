package sms

import (
	"github.com/garyburd/redigo/redis"
	"github.com/uber-go/zap"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func buildTestRedisPool() *redis.Pool {
	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 10 * time.Second,
		Dial: func() (c redis.Conn, err error) {
			c, err = redis.DialTimeout("tcp", "127.0.0.1:6379", 5*time.Second, 3*time.Second, 5*time.Second)
			if err != nil {
				return
			}
			return
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		Wait: false,
	}
	return pool
}

// RateLimitFilterRedis

func TestRateLimitFilterRedis_Filter(t *testing.T) {
	pool := buildTestRedisPool()
	defer pool.Close()

	filter := NewRateLimitFilterRedis(pool, 3, 1)
	ctx := &Context{
		Logger: zap.NewJSON(),
	}
	ctx.Logger.SetLevel(zap.DebugLevel)

	for i := 0; i < 5; i++ {
		resp := &SMSResp{}
		req := &SMSReq{
			PhoneNumbers: []string{"123"},
			Values: [][]string{
				[]string{"v1"},
			},
		}
		filter.Filter(ctx, req, resp)
		t.Logf("req: %#v, resp: %#v", req, resp)
	}
}

func TestRateLimitFilterRedis_Concurrence(t *testing.T) {
	pool := buildTestRedisPool()
	defer pool.Close()

	filter := NewRateLimitFilterRedis(pool, 3, 1)

	start := time.Now()
	sucCount := int64(0)
	wg := &sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			ctx := &Context{
				Logger: zap.NewJSON().With(zap.Int("index", i)),
			}
			ctx.Logger.SetLevel(zap.InfoLevel)

			for i := 0; i < 500; i++ {
				resp := &SMSResp{}
				req := &SMSReq{
					PhoneNumbers: []string{"123"},
					Values: [][]string{
						[]string{"v1"},
					},
				}
				filter.Filter(ctx, req, resp)
				if len(req.PhoneNumbers) > 0 {
					atomic.AddInt64(&sucCount, 1)
				}
			}
		}(i)
	}
	wg.Wait()

	t.Logf("success count: %d, elapse %s", sucCount, time.Now().Sub(start).String())
}

// 116709 ns/op
func BenchmarkRateLimitFilterRedis_Filter(b *testing.B) {
	pool := buildTestRedisPool()
	defer pool.Close()

	filter := NewRateLimitFilterRedis(pool, 1, 1)
	ctx := &Context{
		Logger: zap.NewJSON(),
	}
	ctx.Logger.SetLevel(zap.InfoLevel)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp := &SMSResp{}
		req := &SMSReq{
			PhoneNumbers: []string{"123"},
			Values: [][]string{
				[]string{"v1"},
			},
		}
		filter.Filter(ctx, req, resp)
	}
}

// RateLimitFilterRedisCounter

func TestRateLimitFilterRedisCounter_Filter(t *testing.T) {
	pool := buildTestRedisPool()
	defer pool.Close()

	filter := NewRateLimitFilterRedisCounter(pool, 3, 1, func() string { return strconv.FormatInt(time.Now().Unix(), 10) })
	ctx := &Context{
		Logger: zap.NewJSON(),
	}
	ctx.Logger.SetLevel(zap.DebugLevel)

	for i := 0; i < 5; i++ {
		resp := &SMSResp{}
		req := &SMSReq{
			PhoneNumbers: []string{"1234"},
			Values: [][]string{
				[]string{"v1"},
			},
		}
		filter.Filter(ctx, req, resp)
		t.Logf("req: %#v, resp: %#v", req, resp)
	}
}

func TestRateLimitFilterRedisCounter_Concurrence(t *testing.T) {
	pool := buildTestRedisPool()
	defer pool.Close()

	filter := NewRateLimitFilterRedisCounter(pool, 3, 1, func() string { return strconv.FormatInt(time.Now().Unix(), 10) })

	start := time.Now()
	sucCount := int64(0)
	wg := &sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			ctx := &Context{
				Logger: zap.NewJSON().With(zap.Int("index", i)),
			}
			ctx.Logger.SetLevel(zap.DebugLevel)

			for i := 0; i < 10; i++ {
				resp := &SMSResp{}
				req := &SMSReq{
					PhoneNumbers: []string{"123"},
					Values: [][]string{
						[]string{"v1"},
					},
				}
				filter.Filter(ctx, req, resp)
				if len(req.PhoneNumbers) > 0 {
					atomic.AddInt64(&sucCount, 1)
				}
			}
		}(i)
	}
	wg.Wait()

	t.Logf("success count: %d, elapse %s", sucCount, time.Now().Sub(start).String())
}

// 53577 ns/op
func BenchmarkRateLimitFilterRedisCounter_Filter(b *testing.B) {
	pool := buildTestRedisPool()
	defer pool.Close()

	filter := NewRateLimitFilterRedisCounter(pool, 3, 1, func() string { return strconv.FormatInt(time.Now().Unix(), 10) })
	ctx := &Context{
		Logger: zap.NewJSON(),
	}
	ctx.Logger.SetLevel(zap.InfoLevel)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp := &SMSResp{}
		req := &SMSReq{
			PhoneNumbers: []string{"123"},
			Values: [][]string{
				[]string{"v1"},
			},
		}
		filter.Filter(ctx, req, resp)
	}
}
