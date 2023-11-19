package proxy

import (
	sc "context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"strconv"
	"sync"
	"time"
)

var (
	sharedHeaderLockersValue = []byte("lockers")
)

type LockParam struct {
	Key []byte        `json:"key"`
	TTL time.Duration `json:"ttl"`
}

type LockResult struct {
	Id    []byte          `json:"id"`
	Error json.RawMessage `json:"error"`
}

type LockStatusParam struct {
	Id []byte `json:"id"`
}

type LockStatusResult struct {
	Pass  bool            `json:"pass"`
	Error json.RawMessage `json:"error"`
}

type UnlockParam struct {
	Key []byte `json:"key"`
	Id  []byte `json:"id"`
}

type UnlockResult struct {
	Error json.RawMessage `json:"error"`
}

type Locker struct {
	id        []byte
	key       []byte
	ttl       time.Duration
	client    transports.Client
	signature signatures.Signature
}

func (locker *Locker) status(ctx sc.Context, id []byte) (passed bool, err error) {
	// param
	param := LockStatusParam{
		Id: id,
	}
	p, _ := json.Marshal(param)
	command := Command{
		Command: "status",
		Payload: p,
	}

	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	header.Set(sharedHeader, sharedHeaderLockersValue)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), locker.signature.Sign(body))

	// do
	status, _, responseBody, doErr := locker.client.Do(ctx, transports.MethodPost, sharedHandlerPath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development locker status failed").WithCause(doErr)
		return
	}
	if status == 200 {
		result := LockStatusResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			err = errors.Warning("fns: development locker status failed").WithCause(decodeErr)
			return
		}
		if len(result.Error) > 0 {
			err = errors.Decode(result.Error)
			return
		}
		passed = result.Pass
		return
	}
	err = errors.Warning("fns: development locker status failed").WithMeta("status", strconv.Itoa(status))
	return
}

func (locker *Locker) Lock(ctx sc.Context) (err error) {
	// param
	param := LockParam{
		Key: locker.key,
		TTL: locker.ttl,
	}
	p, _ := json.Marshal(param)
	command := Command{
		Command: "lock",
		Payload: p,
	}
	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	header.Set(sharedHeader, sharedHeaderLockersValue)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), locker.signature.Sign(body))

	// do
	status, _, responseBody, doErr := locker.client.Do(ctx, transports.MethodPost, sharedHandlerPath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development locker lock failed").WithCause(doErr)
		return
	}
	if status == 200 {
		result := LockResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			err = errors.Warning("fns: development locker lock failed").WithCause(decodeErr)
			return
		}
		if len(result.Error) > 0 {
			err = errors.Decode(result.Error)
			return
		}
		locker.id = result.Id
		stop := false
		timer := time.NewTimer(locker.ttl)
		for {
			select {
			case <-timer.C:
				stop = true
				break
			default:
				passed, statusErr := locker.status(ctx, result.Id)
				if statusErr != nil {
					err = errors.Warning("fns: development locker lock failed").WithCause(statusErr)
					return
				}
				if passed {
					stop = true
					break
				}
			}
			if stop {
				timer.Stop()
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		return
	}
	err = errors.Warning("fns: development locker lock failed").WithMeta("status", strconv.Itoa(status))
	return
}

func (locker *Locker) Unlock(ctx sc.Context) (err error) {
	// param
	param := UnlockParam{
		Key: locker.key,
		Id:  locker.id,
	}
	p, _ := json.Marshal(param)
	command := Command{
		Command: "unlock",
		Payload: p,
	}

	body, _ := json.Marshal(command)

	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	header.Set(sharedHeader, sharedHeaderLockersValue)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), locker.signature.Sign(body))

	// do
	status, _, responseBody, doErr := locker.client.Do(ctx, transports.MethodPost, sharedHandlerPath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development locker unlock failed").WithCause(doErr)
		return
	}
	if status == 200 {
		result := UnlockResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			err = errors.Warning("fns: development locker unlock failed").WithCause(decodeErr)
			return
		}
		if len(result.Error) > 0 {
			err = errors.Decode(result.Error)
			return
		}
		return
	}
	err = errors.Warning("fns: development locker unlock failed").WithMeta("status", strconv.Itoa(status))
	return
}

type Lockers struct {
	client    transports.Client
	signature signatures.Signature
}

func (lockers *Lockers) Acquire(_ sc.Context, key []byte, ttl time.Duration) (locker shareds.Locker, err error) {
	locker = &Locker{
		key:       key,
		ttl:       ttl,
		client:    lockers.client,
		signature: lockers.signature,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type LockerStatus struct {
	locker shareds.Locker
	passed bool
	Err    error
}

func NewSharedLockersHandler(lockers shareds.Lockers) transports.Handler {
	return &SharedLockersHandler{
		lockers:   lockers,
		statusMap: sync.Map{},
	}
}

type SharedLockersHandler struct {
	lockers   shareds.Lockers
	signature signatures.Signature
	statusMap sync.Map
}

func (handler *SharedLockersHandler) Handle(w transports.ResponseWriter, r transports.Request) {
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
	case "lock":
		param := LockParam{}
		paramErr := json.Unmarshal(cmd.Payload, &param)
		if paramErr != nil {
			w.Failed(ErrInvalidBody.WithCause(paramErr))
			return
		}
		result := LockResult{}
		locker, lockerErr := handler.lockers.Acquire(r, param.Key, param.TTL)
		if lockerErr != nil {
			result.Error, _ = json.Marshal(errors.Map(lockerErr))
			w.Succeed(result)
			return
		}
		id := uid.Bytes()
		status := LockerStatus{
			locker: locker,
			passed: false,
			Err:    nil,
		}
		handler.statusMap.Store(bytex.ToString(id), status)

		go func(ctx context.Context, statusMap *sync.Map, locker shareds.Locker, id []byte) {
			lockErr := locker.Lock(ctx)
			v, has := statusMap.Load(id)
			if !has {
				return
			}
			ls := v.(LockerStatus)
			ls.passed = lockErr == nil
			ls.Err = lockerErr
			statusMap.Store(id, ls)
		}(r, &handler.statusMap, locker, id)

		result.Id = id
		w.Succeed(result)
		break
	case "status":
		param := LockStatusParam{}
		paramErr := json.Unmarshal(cmd.Payload, &param)
		if paramErr != nil {
			w.Failed(ErrInvalidBody.WithCause(paramErr))
			return
		}
		result := LockStatusResult{}
		v, has := handler.statusMap.Load(bytex.ToString(param.Id))
		if !has {
			result.Error, _ = json.Marshal(errors.Warning("fns: locker status was not found"))
			w.Succeed(result)
			return
		}
		ls := v.(LockerStatus)
		result.Pass = ls.passed
		if ls.Err != nil {
			result.Error, _ = json.Marshal(ls.Err)
		}
		w.Succeed(result)
		break
	case "unlock":
		param := UnlockParam{}
		paramErr := json.Unmarshal(cmd.Payload, &param)
		if paramErr != nil {
			w.Failed(ErrInvalidBody.WithCause(paramErr))
			return
		}
		result := UnlockResult{}
		v, has := handler.statusMap.Load(bytex.ToString(param.Id))
		if !has {
			result.Error, _ = json.Marshal(errors.Warning("fns: locker status was not found"))
			w.Succeed(result)
			return
		}
		ls := v.(LockerStatus)
		handler.statusMap.Delete(bytex.ToString(param.Id))
		unlockErr := ls.locker.Unlock(r)
		if unlockErr != nil {
			result.Error, _ = json.Marshal(unlockErr)
			w.Succeed(result)
			return
		}
		w.Succeed(result)
		break
	default:
		w.Failed(ErrInvalidBody)
		return
	}
}
