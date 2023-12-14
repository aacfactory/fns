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

package authorizations

import (
	"github.com/aacfactory/fns/services"
)

var (
	endpointName = []byte("authorizations")
)

type Options struct {
	encoder TokenEncoder
	store   TokenStore
}

type Option func(options *Options)

func WithTokenEncoder(encoder TokenEncoder) Option {
	return func(options *Options) {
		options.encoder = encoder
	}
}

func WithTokenStore(store TokenStore) Option {
	return func(options *Options) {
		options.store = store
	}
}

func New(options ...Option) services.Service {
	opt := Options{
		encoder: &defaultTokenEncoder{},
		store:   &defaultTokenStore{},
	}
	for _, option := range options {
		option(&opt)
	}
	return &service{
		Abstract: services.NewAbstract(string(endpointName), true, opt.encoder, opt.store),
		encoder:  opt.encoder,
		store:    opt.store,
	}
}

// service
// use @authorization
type service struct {
	services.Abstract
	encoder TokenEncoder
	store   TokenStore
}

func (svc *service) Construct(options services.Options) (err error) {
	err = svc.Abstract.Construct(options)
	if err != nil {
		return
	}
	svc.AddFunction(&encodeFn{
		encoder: svc.encoder,
		store:   svc.store,
	})
	svc.AddFunction(&decodeFn{
		encoder: svc.encoder,
		store:   svc.store,
	})
	svc.AddFunction(&listFn{
		store: svc.store,
	})
	svc.AddFunction(&removeFn{
		store: svc.store,
	})
	return
}
