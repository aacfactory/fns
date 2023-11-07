package development

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"strconv"
	"time"
)

var (
	shardHandleStorePath = []byte("/development/shared/store")
)

type StoreCommand struct {
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

type Store struct {
	address   []byte
	dialer    transports.Dialer
	signature signatures.Signature
}

type StoreGetParam struct {
	Key []byte `json:"key"`
}

type StoreGetResult struct {
	Value []byte          `json:"value"`
	Has   bool            `json:"has"`
	Error json.RawMessage `json:"error"`
}

func (store *Store) Get(ctx context.Context, key []byte) (value []byte, has bool, err error) {
	client, clientErr := store.dialer.Dial(store.address)
	if clientErr != nil {
		err = clientErr
		return
	}
	defer client.Close()
	// param
	param := StoreGetParam{
		Key: key,
	}
	p, _ := json.Marshal(param)
	command := StoreCommand{
		Command: "get",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := client.Do(ctx, methodPost, shardHandleStorePath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development store get failed").WithCause(doErr)
		return
	}
	if status == 200 {
		result := StoreGetResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			err = errors.Warning("fns: development store get failed").WithCause(decodeErr)
			return
		}
		if len(result.Error) > 0 {
			err = errors.Decode(result.Error)
			return
		}
		value = result.Value
		has = result.Has
		return
	}
	err = errors.Warning("fns: development store get failed").WithMeta("status", strconv.Itoa(status))
	return
}

type StoreSetParam struct {
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

type StoreSetResult struct {
	Error json.RawMessage `json:"error"`
}

func (store *Store) Set(ctx context.Context, key []byte, value []byte) (err error) {
	client, clientErr := store.dialer.Dial(store.address)
	if clientErr != nil {
		err = clientErr
		return
	}
	defer client.Close()
	// param
	param := StoreSetParam{
		Key:   key,
		Value: value,
	}
	p, _ := json.Marshal(param)
	command := StoreCommand{
		Command: "set",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := client.Do(ctx, methodPost, shardHandleStorePath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development store set failed").WithCause(doErr)
		return
	}
	if status == 200 {
		result := StoreSetResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			err = errors.Warning("fns: development store set failed").WithCause(decodeErr)
			return
		}
		if len(result.Error) > 0 {
			err = errors.Decode(result.Error)
			return
		}
		return
	}
	err = errors.Warning("fns: development store set failed").WithMeta("status", strconv.Itoa(status))
	return
}

type StoreSetWithTTLParam struct {
	Key   []byte        `json:"key"`
	Value []byte        `json:"value"`
	TTL   time.Duration `json:"ttl"`
}

type StoreSetWithTTLResult struct {
	Error json.RawMessage `json:"error"`
}

func (store *Store) SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	client, clientErr := store.dialer.Dial(store.address)
	if clientErr != nil {
		err = clientErr
		return
	}
	defer client.Close()
	// param
	param := StoreSetWithTTLParam{
		Key:   key,
		Value: value,
		TTL:   ttl,
	}
	p, _ := json.Marshal(param)
	command := StoreCommand{
		Command: "setWithTTL",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := client.Do(ctx, methodPost, shardHandleStorePath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development store set with ttl failed").WithCause(doErr)
		return
	}
	if status == 200 {
		result := StoreSetWithTTLResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			err = errors.Warning("fns: development store set with ttl failed").WithCause(decodeErr)
			return
		}
		if len(result.Error) > 0 {
			err = errors.Decode(result.Error)
			return
		}
		return
	}
	err = errors.Warning("fns: development store set with ttl failed").WithMeta("status", strconv.Itoa(status))
	return
}

type StoreIncrParam struct {
	Key   []byte `json:"key"`
	Delta int64  `json:"delta"`
}

type StoreIncrResult struct {
	N     int64           `json:"n"`
	Error json.RawMessage `json:"error"`
}

func (store *Store) Incr(ctx context.Context, key []byte, delta int64) (v int64, err error) {
	client, clientErr := store.dialer.Dial(store.address)
	if clientErr != nil {
		err = clientErr
		return
	}
	defer client.Close()
	// param
	param := StoreIncrParam{
		Key:   key,
		Delta: delta,
	}
	p, _ := json.Marshal(param)
	command := StoreCommand{
		Command: "incr",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := client.Do(ctx, methodPost, shardHandleStorePath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development store incr failed").WithCause(doErr)
		return
	}
	if status == 200 {
		result := StoreIncrResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			err = errors.Warning("fns: development store incr failed").WithCause(decodeErr)
			return
		}
		if len(result.Error) > 0 {
			err = errors.Decode(result.Error)
			return
		}
		v = result.N
		return
	}
	err = errors.Warning("fns: development store incr failed").WithMeta("status", strconv.Itoa(status))
	return
}

type StoreRemoveParam struct {
	Key []byte `json:"key"`
}

type StoreRemoveResult struct {
	Error json.RawMessage `json:"error"`
}

func (store *Store) Remove(ctx context.Context, key []byte) (err error) {
	client, clientErr := store.dialer.Dial(store.address)
	if clientErr != nil {
		err = clientErr
		return
	}
	defer client.Close()
	// param
	param := StoreRemoveParam{
		Key: key,
	}
	p, _ := json.Marshal(param)
	command := StoreCommand{
		Command: "remove",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := client.Do(ctx, methodPost, shardHandleStorePath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development store remove failed").WithCause(doErr)
		return
	}
	if status == 200 {
		result := StoreRemoveResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			err = errors.Warning("fns: development store remove failed").WithCause(decodeErr)
			return
		}
		if len(result.Error) > 0 {
			err = errors.Decode(result.Error)
			return
		}
		return
	}
	err = errors.Warning("fns: development store remove failed").WithMeta("status", strconv.Itoa(status))
	return
}

func (store *Store) Close() {}

// +-------------------------------------------------------------------------------------------------------------------+

func NewSharedStoreHandler(store shareds.Store, signature signatures.Signature) transports.Handler {
	return &SharedStoreHandler{
		store:     store,
		signature: signature,
	}
}

type SharedStoreHandler struct {
	store     shareds.Store
	signature signatures.Signature
}

func (handler *SharedStoreHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody)
		return
	}
	cmd := StoreCommand{}
	decodeErr := json.Unmarshal(body, &cmd)
	if decodeErr != nil {
		w.Failed(ErrInvalidBody.WithCause(decodeErr))
		return
	}
	switch cmd.Command {
	case "get":
		param := StoreGetParam{}
		paramErr := json.Unmarshal(cmd.Payload, &param)
		if paramErr != nil {
			w.Failed(ErrInvalidBody.WithCause(paramErr))
			return
		}
		result := StoreGetResult{}
		value, has, err := handler.store.Get(r, param.Key)
		if err == nil {
			result.Value = value
			result.Has = has
		} else {
			result.Error, _ = json.Marshal(errors.Map(err))
		}
		w.Succeed(result)
		break
	case "set":
		param := StoreSetParam{}
		paramErr := json.Unmarshal(cmd.Payload, &param)
		if paramErr != nil {
			w.Failed(ErrInvalidBody.WithCause(paramErr))
			return
		}
		result := StoreSetResult{}
		err := handler.store.Set(r, param.Key, param.Value)
		if err == nil {
		} else {
			result.Error, _ = json.Marshal(errors.Map(err))
		}
		w.Succeed(result)
		break
	case "setWithTTL":
		param := StoreSetWithTTLParam{}
		paramErr := json.Unmarshal(cmd.Payload, &param)
		if paramErr != nil {
			w.Failed(ErrInvalidBody.WithCause(paramErr))
			return
		}
		result := StoreSetWithTTLResult{}
		err := handler.store.SetWithTTL(r, param.Key, param.Value, param.TTL)
		if err == nil {
		} else {
			result.Error, _ = json.Marshal(errors.Map(err))
		}
		w.Succeed(result)
		break
	case "incr":
		param := StoreIncrParam{}
		paramErr := json.Unmarshal(cmd.Payload, &param)
		if paramErr != nil {
			w.Failed(ErrInvalidBody.WithCause(paramErr))
			return
		}
		result := StoreIncrResult{}
		n, err := handler.store.Incr(r, param.Key, param.Delta)
		if err == nil {
			result.N = n
		} else {
			result.Error, _ = json.Marshal(errors.Map(err))
		}
		w.Succeed(result)
		break
	case "remove":
		param := StoreRemoveParam{}
		paramErr := json.Unmarshal(cmd.Payload, &param)
		if paramErr != nil {
			w.Failed(ErrInvalidBody.WithCause(paramErr))
			return
		}
		result := StoreRemoveResult{}
		err := handler.store.Remove(r, param.Key)
		if err == nil {
		} else {
			result.Error, _ = json.Marshal(errors.Map(err))
		}
		w.Succeed(result)
		break
	default:
		w.Failed(ErrInvalidBody)
		return
	}
}
