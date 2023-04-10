/*
 * Copyright 2021 Wang Min Xiang
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
 */

package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/commons/wildcard"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"net/http"
	"strconv"
	"strings"
)

var (
	allowAllRequestVersions = RequestVersions{
		{
			Pattern: "*",
			Begin:   versions.Origin(),
			End:     versions.Latest(),
			Exact:   false,
		},
	}
)

func AllowAllRequestVersions() RequestVersions {
	return allowAllRequestVersions
}

type RequestVersions []*RequestVersion

func (rvs RequestVersions) Accept(serviceName string, target versions.Version) (ok bool) {
	if rvs == nil || len(rvs) == 0 {
		return
	}
	for _, rv := range rvs {
		if rv == nil {
			continue
		}
		if rv.Accept(serviceName, target) {
			return true
		}
	}
	return
}

func (rvs RequestVersions) Empty() (ok bool) {
	ok = rvs == nil || len(rvs) == 0
	return
}

func (rvs RequestVersions) String() string {
	if rvs == nil {
		return "nil"
	}
	ss := make([]string, 0, 1)
	for _, rv := range rvs {
		if rv == nil {
			continue
		}
		ss = append(ss, rv.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(ss, ", "))
}

func ParseRequestVersionFromHeader(header transports.Header) (rvs RequestVersions, has bool, err error) {
	values := header.Values(httpRequestVersionsHeader)
	if values == nil || len(values) == 0 {
		return
	}
	rvs = make([]*RequestVersion, 0, 1)
	for _, value := range values {
		if value == "" {
			continue
		}
		rv, parseErr := ParseRequestVersion(value)
		if parseErr != nil {
			err = parseErr
			return
		}
		rvs = append(rvs, rv)
	}
	has = len(rvs) > 0
	return
}

func ParseRequestVersion(s string) (rv *RequestVersion, err error) {
	if s == "" {
		err = errors.Warning("fns: parse request version failed").WithCause(errors.Warning("value is nil"))
		return
	}
	idx := strings.Index(s, "=")
	if idx < 1 {
		err = errors.Warning("fns: parse request version failed").WithCause(errors.Warning("no pattern"))
		return
	}
	pattern := strings.TrimSpace(s[0:idx])
	s = s[idx+1:]
	idx = strings.Index(s, ":")
	if idx < 0 {
		beg, parseBegErr := versions.Parse(s)
		if parseBegErr != nil {
			err = errors.Warning("fns: parse request version failed").WithCause(parseBegErr)
			return
		}
		rv = &RequestVersion{
			Pattern: pattern,
			Begin:   beg,
			End:     versions.Version{},
			Exact:   true,
		}
		return
	}
	beg := versions.Origin()
	begValue := strings.TrimSpace(s[0:idx])
	if begValue != "" {
		beg, err = versions.Parse(begValue)
		if err != nil {
			err = errors.Warning("fns: parse request version failed").WithCause(err)
			return
		}
	}
	end := versions.Latest()
	endValue := strings.TrimSpace(s[idx+1:])
	if endValue != "" {
		end, err = versions.Parse(endValue)
		if err != nil {
			err = errors.Warning("fns: parse request version failed").WithCause(err)
			return
		}
	}
	rv = &RequestVersion{
		Pattern: pattern,
		Begin:   beg,
		End:     end,
		Exact:   false,
	}
	return
}

type RequestVersion struct {
	Pattern string
	Begin   versions.Version
	End     versions.Version
	Exact   bool
}

func (rv *RequestVersion) Accept(serviceName string, target versions.Version) (ok bool) {
	if !wildcard.Match(rv.Pattern, serviceName) {
		return
	}
	if rv.Exact {
		ok = rv.Begin.Equals(target)
		return
	}
	ok = target.Between(rv.Begin, rv.End)
	return
}

func (rv *RequestVersion) String() string {
	if rv.Exact {
		return fmt.Sprintf("%s=%s", rv.Pattern, rv.Begin.String())
	}
	return fmt.Sprintf("%s=%s:%s", rv.Pattern, rv.Begin.String(), rv.End.String())
}

type RequestHeader http.Header

func (header RequestHeader) Empty() bool {
	return header == nil || len(header) == 0
}

func (header RequestHeader) Add(key, value string) {
	http.Header(header).Add(key, value)
}

func (header RequestHeader) Set(key, value string) {
	http.Header(header).Set(key, value)
}

func (header RequestHeader) Get(key string) string {
	return http.Header(header).Get(key)
}

func (header RequestHeader) Values(key string) []string {
	return http.Header(header).Values(key)
}

func (header RequestHeader) Del(key string) {
	http.Header(header).Del(key)
}

func (header RequestHeader) Clone() RequestHeader {
	return RequestHeader(http.Header(header).Clone())
}

func (header RequestHeader) Contains(key string) (ok bool) {
	ok = http.Header(header).Get(key) != ""
	return
}

func (header RequestHeader) Authorization() (authorization string, has bool) {
	authorization = http.Header(header).Get("Authorization")
	has = authorization != ""
	return
}

func (header RequestHeader) SetDeviceId(id string) {
	http.Header(header).Set(httpDeviceIdHeader, id)
	return
}

func (header RequestHeader) DeviceId() (id string) {
	id = http.Header(header).Get(httpDeviceIdHeader)
	return
}

func (header RequestHeader) SetDeviceIp(ip string) {
	http.Header(header).Set(httpDeviceIpHeader, ip)
	return
}

func (header RequestHeader) DeviceIp() (id string) {
	id = http.Header(header).Get(httpDeviceIpHeader)
	return
}

func (header RequestHeader) SetAcceptVersions(rvs RequestVersions) {
	header.Del(httpRequestVersionsHeader)
	if rvs == nil || len(rvs) == 0 {
		return
	}
	for _, rv := range rvs {
		if rv == nil {
			return
		}
		header.Add(httpRequestVersionsHeader, rv.String())
	}
	return
}

func (header RequestHeader) CacheControlDisabled() (ok bool) {
	cc := bytex.FromString(http.Header(header).Get(httpCacheControlHeader))
	ok = bytes.Contains(cc, bytex.FromString(httpCacheControlNoStore)) || bytes.Contains(cc, bytex.FromString(httpCacheControlNoCache))
	return
}

func (header RequestHeader) DisableCacheControl() {
	header.Set(httpCacheControlHeader, httpCacheControlNoStore)
	return
}

func (header RequestHeader) EnableCacheControl(etag []byte) {
	header.Set(httpCacheControlHeader, httpCacheControlEnabled)
	header.Set(httpCacheControlIfNonMatch, bytex.ToString(etag))
	return
}

func (header RequestHeader) ForEach(fn func(key string, values []string) (next bool)) {
	if fn == nil {
		return
	}
	for key, values := range header {
		next := fn(key, values)
		if !next {
			break
		}
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

type RequestUserId string

func (id RequestUserId) Int() (n int64) {
	s := string(id)
	v, parseErr := strconv.ParseInt(s, 10, 64)
	if parseErr != nil {
		panic(errors.Warning(fmt.Sprintf("fns: parse user id to int failed")).WithMeta("scope", "system").WithMeta("id", s).WithCause(parseErr))
	}
	n = v
	return
}

func (id RequestUserId) String() string {
	return string(id)
}

func (id RequestUserId) Exist() (ok bool) {
	ok = id != "" && id != "0"
	return
}

type RequestUser interface {
	json.Marshaler
	json.Unmarshaler
	Authenticated() (ok bool)
	Id() (id RequestUserId)
	SetId(id RequestUserId)
	Attributes() (attributes *json.Object)
	SetAttributes(attributes *json.Object)
}

func NewRequestUser(id RequestUserId, attributes *json.Object) (u RequestUser) {
	if attributes == nil {
		attributes = json.NewObject()
	}
	u = &requestUser{
		id:         id,
		attributes: attributes,
	}
	return
}

type requestUser struct {
	id         RequestUserId
	attributes *json.Object
}

func (u *requestUser) Authenticated() (ok bool) {
	ok = u.id != ""
	return
}

func (u *requestUser) Id() (id RequestUserId) {
	id = u.id
	return
}

func (u *requestUser) SetId(id RequestUserId) {
	u.id = id
}

func (u *requestUser) Attributes() (attributes *json.Object) {
	attributes = u.attributes
	return
}

func (u *requestUser) SetAttributes(attributes *json.Object) {
	if attributes == nil {
		attributes = json.NewObject()
	}
	u.attributes = attributes
}

func (u *requestUser) MarshalJSON() (p []byte, err error) {
	o := json.NewObject()
	_ = o.Put("id", u.id)
	_ = o.Put("authenticated", u.Authenticated())
	if u.attributes == nil {
		u.attributes = json.NewObject()
	}
	_ = o.Put("attributes", u.attributes)
	p, err = o.MarshalJSON()
	return
}

func (u *requestUser) UnmarshalJSON(p []byte) (err error) {
	o := json.NewObjectFromBytes(p)
	err = o.Get("id", &u.id)
	if err != nil {
		return
	}
	if u.attributes == nil {
		u.attributes = json.NewObject()
	}
	err = o.Get("attributes", &u.attributes)
	if err != nil {
		return
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type RequestTrunk interface {
	json.Marshaler
	json.Unmarshaler
	Get(key string) (value []byte, has bool)
	Put(key string, value []byte)
	ForEach(fn func(key string, value []byte) (next bool))
	ReadFrom(o RequestTrunk)
	Remove(key string)
}

func newRequestTrunk() RequestTrunk {
	return &requestTrunk{
		values: make(map[string][]byte),
	}
}

type requestTrunk struct {
	values map[string][]byte
}

func (trunk *requestTrunk) MarshalJSON() (p []byte, err error) {
	p, err = json.Marshal(trunk.values)
	return
}

func (trunk *requestTrunk) UnmarshalJSON(p []byte) (err error) {
	values := make(map[string][]byte)
	err = json.Unmarshal(p, &values)
	if err != nil {
		return
	}
	trunk.values = values
	return
}

func (trunk *requestTrunk) ReadFrom(o RequestTrunk) {
	trunk.values = make(map[string][]byte)
	o.ForEach(func(key string, value []byte) (next bool) {
		trunk.values[key] = value
		next = true
		return
	})
	return
}

func (trunk *requestTrunk) Get(key string) (value []byte, has bool) {
	value, has = trunk.values[key]
	return
}

func (trunk *requestTrunk) Put(key string, value []byte) {
	trunk.values[key] = value
}

func (trunk *requestTrunk) ForEach(fn func(key string, value []byte) (next bool)) {
	if fn == nil {
		return
	}
	for ket, value := range trunk.values {
		next := fn(ket, value)
		if !next {
			break
		}
	}
}

func (trunk *requestTrunk) Remove(key string) {
	delete(trunk.values, key)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Request interface {
	Id() (id string)
	Header() (header RequestHeader)
	AcceptedVersions() (acceptedVersions RequestVersions)
	Fn() (service string, fn string)
	Argument() (argument Argument)
	Internal() (ok bool)
	User() (user RequestUser)
	Trunk() (trunk RequestTrunk)
	Hash() (p []byte)
}

type RequestOption func(*RequestOptions)

func WithRequestId(id string) RequestOption {
	return func(options *RequestOptions) {
		options.id = id
	}
}

func WithRequestHeader(header transports.Header) RequestOption {
	return func(options *RequestOptions) {
		options.header = RequestHeader(header)
	}
}

func WithDeviceId(id string) RequestOption {
	return func(options *RequestOptions) {
		options.deviceId = id
	}
}

func WithDeviceIp(ip string) RequestOption {
	return func(options *RequestOptions) {
		options.deviceIp = ip
	}
}

func WithInternalRequest() RequestOption {
	return func(options *RequestOptions) {
		options.internal = true
	}
}

func WithRequestUser(id RequestUserId, attributes *json.Object) RequestOption {
	return func(options *RequestOptions) {
		options.user = NewRequestUser(id, attributes)
	}
}

func WithRequestTrunk(trunk RequestTrunk) RequestOption {
	return func(options *RequestOptions) {
		options.trunk = trunk
	}
}

func WithRequestVersions(acceptedVersions RequestVersions) RequestOption {
	return func(options *RequestOptions) {
		options.acceptedVersions = acceptedVersions
	}
}

func DisableRequestCacheControl() RequestOption {
	return func(options *RequestOptions) {
		options.disableCacheControl = true
	}
}

type RequestOptions struct {
	id                  string
	header              RequestHeader
	acceptedVersions    RequestVersions
	deviceId            string
	deviceIp            string
	internal            bool
	user                RequestUser
	trunk               RequestTrunk
	disableCacheControl bool
}

func NewRequest(ctx context.Context, service string, fn string, arg Argument, options ...RequestOption) (v Request) {
	opt := &RequestOptions{
		id:                  "",
		header:              RequestHeader{},
		acceptedVersions:    nil,
		deviceId:            "",
		deviceIp:            "",
		user:                nil,
		internal:            false,
		trunk:               nil,
		disableCacheControl: false,
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	if arg == nil {
		arg = EmptyArgument()
	}
	prev, hasPrev := GetRequest(ctx)
	if hasPrev {
		id := prev.Id()
		if opt.id != "" {
			id = opt.id
		}
		var header RequestHeader
		if !opt.header.Empty() {
			header = opt.header
		} else {
			header = prev.Header().Clone()
			header.Del(httpRequestVersionsHeader)
			header.Del(httpCacheControlHeader)
			header.Del(httpCacheControlIfNonMatch)
		}
		if opt.deviceId != "" {
			header.SetDeviceId(opt.deviceId)
		}
		if opt.deviceIp != "" {
			header.SetDeviceIp(opt.deviceIp)
		}
		if opt.disableCacheControl {
			header.DisableCacheControl()
		}
		user := prev.User()
		if opt.user != nil {
			user = opt.user
		}
		trunk := prev.Trunk()
		if opt.trunk != nil {
			trunk = opt.trunk
		}
		acceptedVersions := prev.AcceptedVersions()
		if opt.acceptedVersions != nil {
			acceptedVersions = opt.acceptedVersions
		}
		if acceptedVersions == nil {
			acceptedVersions = AllowAllRequestVersions()
		}
		v = &request{
			id:               id,
			internal:         true,
			user:             user,
			trunk:            trunk,
			header:           header,
			service:          service,
			fn:               fn,
			argument:         arg,
			acceptedVersions: acceptedVersions,
		}
	} else {
		id := opt.id
		if id == "" {
			id = uid.UID()
		}
		var header RequestHeader
		if opt.header != nil {
			header = opt.header
		} else {
			header = RequestHeader{}
		}
		if opt.deviceId != "" {
			header.SetDeviceId(opt.deviceId)
		}
		if opt.deviceIp != "" {
			header.SetDeviceIp(opt.deviceIp)
		}
		user := opt.user
		if user == nil {
			user = NewRequestUser("", nil)
		}
		var trunk RequestTrunk
		if opt.trunk != nil {
			trunk = opt.trunk
		} else {
			trunk = newRequestTrunk()
		}
		var acceptedVersions RequestVersions
		if opt.acceptedVersions != nil {
			acceptedVersions = opt.acceptedVersions
		} else {
			acceptedVersions = AllowAllRequestVersions()
		}
		v = &request{
			id:               id,
			internal:         opt.internal,
			user:             user,
			trunk:            trunk,
			header:           header,
			service:          service,
			fn:               fn,
			argument:         arg,
			acceptedVersions: acceptedVersions,
		}
	}
	return
}

type request struct {
	id               string
	internal         bool
	user             RequestUser
	trunk            RequestTrunk
	header           RequestHeader
	acceptedVersions RequestVersions
	service          string
	fn               string
	argument         Argument
	hash             []byte
}

func (r *request) Id() (id string) {
	id = r.id
	return
}

func (r *request) Internal() (ok bool) {
	ok = r.internal
	return
}

func (r *request) User() (user RequestUser) {
	user = r.user
	return
}

func (r *request) Trunk() (trunk RequestTrunk) {
	trunk = r.trunk
	return
}

func (r *request) Header() (header RequestHeader) {
	header = r.header
	return
}

func (r *request) AcceptedVersions() (acceptedVersions RequestVersions) {
	acceptedVersions = r.acceptedVersions
	return
}

func (r *request) Fn() (service string, fn string) {
	service, fn = r.service, r.fn
	return
}

func (r *request) Argument() (argument Argument) {
	argument = r.argument
	return
}

func (r *request) Hash() (p []byte) {
	if len(r.hash) > 0 {
		p = r.hash
		return
	}
	path := bytex.FromString(fmt.Sprintf("/%s/%s", r.service, r.fn))
	body, _ := json.Marshal(r.argument)
	r.hash = requestHash(path, body)
	p = r.hash
	return
}

func requestHash(path []byte, body []byte) (v []byte) {
	buf := bytebufferpool.Get()
	_, _ = buf.Write(path)
	_, _ = buf.Write(body)
	p := buf.Bytes()
	bytebufferpool.Put(buf)
	v = bytex.FromString(strconv.FormatUint(xxhash.Sum64(p), 10))
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	contextRequestKey = "@fns_request"
)

func GetRequest(ctx context.Context) (r Request, has bool) {
	vbv := ctx.Value(contextRequestKey)
	if vbv == nil {
		return
	}
	r, has = vbv.(Request)
	return
}

func withRequest(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, contextRequestKey, r)
}

func GetRequestUser(ctx context.Context) (user RequestUser, authenticated bool) {
	req, hasReq := GetRequest(ctx)
	if !hasReq {
		return
	}
	user = req.User()
	authenticated = user.Authenticated()
	return
}

type internalRequestImpl struct {
	User     *requestUser  `json:"user"`
	Trunk    *requestTrunk `json:"trunk"`
	Argument *argument     `json:"argument"`
}

type internalRequest struct {
	User     RequestUser  `json:"user"`
	Trunk    RequestTrunk `json:"trunk"`
	Argument Argument     `json:"argument,omitempty"`
}

type internalResponseImpl struct {
	User    *requestUser    `json:"user"`
	Trunk   *requestTrunk   `json:"trunk"`
	Span    *Span           `json:"Span"`
	Succeed bool            `json:"succeed"`
	Body    json.RawMessage `json:"body"`
}

type internalResponse struct {
	User    RequestUser  `json:"user"`
	Trunk   RequestTrunk `json:"trunk"`
	Span    *Span        `json:"Span"`
	Succeed bool         `json:"succeed"`
	Body    interface{}  `json:"body,omitempty"`
}
