package development

import (
	"bytes"
	"context"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
)

func NewBarrier(dialer transports.Dialer, address []byte, signature signatures.Signature) barriers.Barrier {
	return &Barrier{
		address:   address,
		dialer:    dialer,
		signature: signature,
		group:     new(singleflight.Group),
	}
}

type Barrier struct {
	address   []byte
	dialer    transports.Dialer
	signature signatures.Signature
	group     *singleflight.Group
}

func (barrier *Barrier) Construct(_ barriers.Options) (err error) {
	return
}

func (barrier *Barrier) Do(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result barriers.Result, err error) {
	//TODO implement me
	panic("implement me")
}

func (barrier *Barrier) Forget(ctx context.Context, key []byte) {
	//TODO implement me
	panic("implement me")
}

// +-------------------------------------------------------------------------------------------------------------------+

var (
	barrierContentType = bytex.FromString("application/json+dev+barrier")
	barrierPathPrefix  = []byte("/barrier/")
)

func NewBarrierHandler(secret string) transports.MuxHandler {
	return &BarrierHandler{
		signature: signatures.HMAC([]byte(secret)),
	}
}

type BarrierHandler struct {
	log       logs.Logger
	signature signatures.Signature
}

func (handler *BarrierHandler) Name() string {
	return "development_barriers"
}

func (handler *BarrierHandler) Construct(options transports.MuxHandlerOptions) error {
	handler.log = options.Log
	return nil
}

func (handler *BarrierHandler) Match(method []byte, path []byte, header transports.Header) bool {
	ok := bytes.Equal(method, methodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 && bytes.LastIndex(path, barrierPathPrefix) == 0 &&
		len(header.Get(bytex.FromString(transports.SignatureHeaderName))) != 0 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), barrierContentType)
	return ok
}

func (handler *BarrierHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	//TODO implement me
	panic("implement me")
}
