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

package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/commons/wildcard"
	"github.com/aacfactory/fns/services/users"
	"github.com/aacfactory/fns/transports"
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

func ParseRequestVersionFromHeader(header http.Header) (rvs RequestVersions, has bool, err error) {
	values := header.Values(transports.RequestVersionsHeaderName)
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
	authorization = http.Header(header).Get(transports.AuthorizationHeaderName)
	has = authorization != ""
	return
}

func (header RequestHeader) SetDeviceId(id string) {
	http.Header(header).Set(transports.DeviceIdHeaderName, id)
	return
}

func (header RequestHeader) DeviceId() (id string) {
	id = http.Header(header).Get(transports.DeviceIdHeaderName)
	return
}

func (header RequestHeader) SetDeviceIp(ip string) {
	http.Header(header).Set(transports.DeviceIpHeaderName, ip)
	return
}

func (header RequestHeader) DeviceIp() (id string) {
	id = http.Header(header).Get(transports.DeviceIpHeaderName)
	return
}

func (header RequestHeader) SetAcceptVersions(rvs RequestVersions) {
	header.Del(transports.RequestVersionsHeaderName)
	if rvs == nil || len(rvs) == 0 {
		return
	}
	for _, rv := range rvs {
		if rv == nil {
			return
		}
		header.Add(transports.RequestVersionsHeaderName, rv.String())
	}
	return
}

func (header RequestHeader) CacheControlDisabled() (ok bool) {
	cc := bytex.FromString(http.Header(header).Get(transports.CacheControlHeaderName))
	ok = bytes.Contains(cc, bytex.FromString(transports.CacheControlHeaderNoStore)) || bytes.Contains(cc, bytex.FromString(transports.CacheControlHeaderNoCache))
	return
}

func (header RequestHeader) DisableCacheControl() {
	header.Set(transports.CacheControlHeaderName, transports.CacheControlHeaderNoStore)
	return
}

func (header RequestHeader) EnableCacheControl(etag []byte) {
	header.Set(transports.CacheControlHeaderName, transports.CacheControlHeaderEnabled)
	header.Set(transports.CacheControlHeaderIfNonMatch, bytex.ToString(etag))
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

type Request interface {
	Id() (id string)
	DeviceId() (id string)
	Header() (header RequestHeader)
	AcceptedVersions() (acceptedVersions RequestVersions)
	Fn() (service string, fn string)
	Argument() (argument Argument)
	Internal() (ok bool)
	User() (user users.User)
	Hash() (p []byte)
}

type RequestOption func(*RequestOptions)

func WithRequestId(id string) RequestOption {
	return func(options *RequestOptions) {
		options.id = id
	}
}

func WithRequestHeader(header http.Header) RequestOption {
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

func WithRequestUser(id RequestUserId, attributes RequestUserAttributes) RequestOption {
	return func(options *RequestOptions) {
		options.user.Id = id
		options.user.Attributes = attributes
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
	disableCacheControl bool
}

func NewRequest(ctx context.Context, service string, fn string, arg Argument, options ...RequestOption) (v Request) {
	opt := &RequestOptions{
		id:                  "",
		header:              RequestHeader{},
		acceptedVersions:    nil,
		deviceId:            "",
		deviceIp:            "",
		user:                NewRequestUser("", NewRequestUserAttributes()),
		internal:            false,
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
			header.Del(httpSignatureHeader)
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
		if opt.user.Authenticated() {
			user = &opt.user
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
		var hash []byte
		if hash0 := header.Get(httpRequestHashHeader); hash0 != "" {
			hash = bytex.FromString(hash0)
			header.Del(httpRequestHashHeader)
		}
		user := opt.user

		var acceptedVersions RequestVersions
		if opt.acceptedVersions != nil {
			acceptedVersions = opt.acceptedVersions
		} else {
			acceptedVersions = AllowAllRequestVersions()
		}
		v = &request{
			id:               id,
			internal:         opt.internal,
			user:             &user,
			header:           header,
			service:          service,
			fn:               fn,
			argument:         arg,
			acceptedVersions: acceptedVersions,
			hash:             hash,
		}
	}
	return
}

type request struct {
	id               string
	internal         bool
	user             *RequestUser
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

func (r *request) User() (user *RequestUser) {
	user = r.user
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
	r.hash = makeRequestHash(path, body)
	p = r.hash
	return
}

func makeRequestHash(path []byte, body []byte) (v []byte) {
	buf := bytebufferpool.Get()
	_, _ = buf.Write(path)
	_, _ = buf.Write(body)
	p := buf.Bytes()
	bytebufferpool.Put(buf)
	v = bytex.FromString(strconv.FormatUint(xxhash.Sum64(p), 10))
	return
}

func getOrMakeRequestHash(header http.Header, path []byte, body []byte) (v []byte) {
	vv := header.Get(httpRequestHashHeader)
	if vv != "" {
		v = bytex.FromString(vv)
		return
	}
	v = makeRequestHash(path, body)
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

func GetRequestUser(ctx context.Context) (user *RequestUser, authenticated bool) {
	req, hasReq := GetRequest(ctx)
	if !hasReq {
		return
	}
	user = req.User()
	authenticated = user.Authenticated()
	return
}
