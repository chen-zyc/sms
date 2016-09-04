package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/uber-go/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"github.com/zhangyuchen0411/sms"
	"github.com/zhangyuchen0411/sms/sms_grpc"
	"time"
)

func SendSMSTest() {
	go func() {
		selector := &sms.RandomSelector{}
		selector.AddSender("test", &sms.MockSender{})

		logger := zap.NewJSON()
		logger.SetLevel(zap.DebugLevel)

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
		rateLimitFilter := sms.NewRateLimitFilterRedis(pool, 5, 3, 10*time.Second)
		rateLimitFilter.KeyExpireSec = 30
		sms.RegisterFilter("test", rateLimitFilter.FilterFunc())

		err := sms_grpc.RunSMSServer(&sms.Context{
			Selector: selector,
			Logger:   logger,
		}, sms_grpc.Options{
			Address: ":8080",
		})

		if err != nil {
			log.Fatal(err)
		}
	}()

	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := sms_grpc.NewSMSSenderClient(conn)

	start := time.Now()
	sucCount := 0

	var testFunc = func(times int) (sucCount int) {
		for i := 0; i < times; i++ {
			resp, err := c.Send(context.Background(), &sms_grpc.SMSReq{
				Category:     "test",
				TemplateID:   "100001",
				PhoneNumbers: []string{"10000"},
			})

			if err != nil {
				fmt.Printf("send err: %s\n", err)
				continue
			}
			if resp.Code != sms.CodeSuccess {
				fmt.Printf("resp not success: %#v\n", resp)
			} else {
				sucCount++
			}
		}
		return
	}

	sucCount += testFunc(10)
	if sucCount != 5 {
		fmt.Printf("send 10 times, success count %d, want 5\n", sucCount)
	}
	time.Sleep(10 * time.Second)
	sucCount += testFunc(10)
	if sucCount != 5+3 {
		fmt.Printf("send 20 times, success count %d, want 8\n", sucCount)
	}

	fmt.Println("finish:", time.Now().Sub(start).String(), ", sucCount:", sucCount)
}
