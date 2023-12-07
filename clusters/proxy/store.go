/*
 * Copyright 2023 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package proxy

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"strconv"
	"time"
)

var (
	sharedHeaderStoreValue = []byte("store")
)

type Store struct {
	client    transports.Client
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
	// param
	param := StoreGetParam{
		Key: key,
	}
	p, _ := json.Marshal(param)
	command := Command{
		Command: "get",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(transports.ContentTypeHeaderName, contentType)
	header.Set(sharedHeader, sharedHeaderStoreValue)
	// signature
	header.Set(transports.SignatureHeaderName, store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := store.client.Do(ctx, transports.MethodPost, sharedHandlerPath, header, body)
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
	// param
	param := StoreSetParam{
		Key:   key,
		Value: value,
	}
	p, _ := json.Marshal(param)
	command := Command{
		Command: "set",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(transports.ContentTypeHeaderName, contentType)
	header.Set(sharedHeader, sharedHeaderStoreValue)
	// signature
	header.Set(transports.SignatureHeaderName, store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := store.client.Do(ctx, transports.MethodPost, sharedHandlerPath, header, body)
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
	// param
	param := StoreSetWithTTLParam{
		Key:   key,
		Value: value,
		TTL:   ttl,
	}
	p, _ := json.Marshal(param)
	command := Command{
		Command: "setWithTTL",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(transports.ContentTypeHeaderName, contentType)
	header.Set(sharedHeader, sharedHeaderStoreValue)
	// signature
	header.Set(transports.SignatureHeaderName, store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := store.client.Do(ctx, transports.MethodPost, sharedHandlerPath, header, body)
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
	// param
	param := StoreIncrParam{
		Key:   key,
		Delta: delta,
	}
	p, _ := json.Marshal(param)
	command := Command{
		Command: "incr",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(transports.ContentTypeHeaderName, contentType)
	header.Set(sharedHeader, sharedHeaderStoreValue)
	// signature
	header.Set(transports.SignatureHeaderName, store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := store.client.Do(ctx, transports.MethodPost, sharedHandlerPath, header, body)
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
	// param
	param := StoreRemoveParam{
		Key: key,
	}
	p, _ := json.Marshal(param)
	command := Command{
		Command: "remove",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(transports.ContentTypeHeaderName, contentType)
	header.Set(sharedHeader, sharedHeaderStoreValue)
	// signature
	header.Set(transports.SignatureHeaderName, store.signature.Sign(body))
	// do
	status, _, responseBody, doErr := store.client.Do(ctx, transports.MethodPost, sharedHandlerPath, header, body)
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

func NewSharedStoreHandler(store shareds.Store) transports.Handler {
	return &SharedStoreHandler{
		store: store,
	}
}

type SharedStoreHandler struct {
	store shareds.Store
}

func (handler *SharedStoreHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody)
		return
	}
	cmd := Command{}
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
			result.Error, _ = json.Marshal(errors.Wrap(err))
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
			result.Error, _ = json.Marshal(errors.Wrap(err))
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
			result.Error, _ = json.Marshal(errors.Wrap(err))
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
			result.Error, _ = json.Marshal(errors.Wrap(err))
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
			result.Error, _ = json.Marshal(errors.Wrap(err))
		}
		w.Succeed(result)
		break
	default:
		w.Failed(ErrInvalidBody)
		return
	}
}
