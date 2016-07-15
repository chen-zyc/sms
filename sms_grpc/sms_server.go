package sms_grpc

import (
	"github.com/uber-go/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"sms"
	"sync"
)

type Options struct {
	Address string
}

type SMSServer struct {
	opt     Options
	ctx     *sms.Context
	reqPool *sync.Pool
}

func RunSMSServer(ctx *sms.Context, opt Options) error {
	if ctx == nil {
		ctx = &sms.Context{}
	}
	if ctx.Logger == nil {
		ctx.Logger = zap.NewJSON()
	}
	if ctx.Selector == nil {
		ctx.Selector = &sms.RandomSelector{}
	}
	if ctx.IDGen == nil {
		var err error
		ctx.IDGen, err = sms.NewDefaultIDGen()
		if err != nil {
			return err
		}
	}

	reqPool := &sync.Pool{
		New: func() interface{} {
			return &sms.SMSReq{}
		},
	}

	lis, err := net.Listen("tcp", opt.Address)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	RegisterSMSSenderServer(s, &SMSServer{
		opt:     opt,
		ctx:     ctx,
		reqPool: reqPool,
	})

	return s.Serve(lis)
}

func (s *SMSServer) Send(ctx context.Context, req *SMSReq) (resp *SMSResp, err error) {
	r := s.reqPool.Get().(*sms.SMSReq)
	r.Category = req.Category
	r.TemplateID = req.TemplateID
	r.PhoneNumbers = req.PhoneNumbers
	r.Args = req.Args

	res := sms.Send(s.ctx, r)

	s.reqPool.Put(r)

	fail := make([]*FailReq, len(res.Fail))
	for i, f := range res.Fail {
		fail[i] = &FailReq{
			PhoneNumber: f.PhoneNumber,
			FailReason:  f.FailReason,
		}
	}
	resp = &SMSResp{
		Code:    res.Code,
		Id:      res.ID,
		Message: res.Message,
		Fail:    fail,
	}

	s.ctx.Logger.Info("send result", zap.Object("req", req), zap.Object("resp", resp))

	return
}
