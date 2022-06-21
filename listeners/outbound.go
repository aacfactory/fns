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

package listeners

import (
	"context"
	"github.com/aacfactory/errors"
	"sync"
)

type MessageHeader interface {
	Get(key string) (value interface{})
	Set(key string, value interface{})
}

type messageHeader map[string]interface{}

func (header messageHeader) Get(key string) (value interface{}) {
	value, _ = header[key]
	return
}

func (header messageHeader) Set(key string, value interface{}) {
	header[key] = value
}

type Message interface {
	Header() (header MessageHeader)
	SetBody(body []byte)
	Body() (body []byte)
}

func NewMessage() Message {
	return &message{
		Header_: messageHeader(make(map[string]interface{})),
		Body_:   nil,
	}
}

type message struct {
	Header_ MessageHeader `json:"header"`
	Body_   []byte        `json:"body"`
}

func (msg *message) Header() (header MessageHeader) {
	header = msg.Header_
	return
}

func (msg *message) SetBody(body []byte) {
	msg.Body_ = body
}

func (msg *message) Body() (body []byte) {
	body = msg.Body_
	return
}

type OutboundChannel interface {
	Name() (name string)
	Send(ctx context.Context, msg Message) (err errors.CodeError)
}

type OutboundChannels interface {
	Get(name string) (channel OutboundChannel, has bool)
	Set(name string, channel OutboundChannel)
}

func NewDefaultOutboundChannels() OutboundChannels {
	return &DefaultOutboundChannels{
		values: sync.Map{},
	}
}

type DefaultOutboundChannels struct {
	values sync.Map
}

func (channels *DefaultOutboundChannels) Get(name string) (channel OutboundChannel, has bool) {
	value, exist := channels.values.Load(name)
	if !exist {
		return
	}
	channel, has = value.(OutboundChannel)
	return
}

func (channels *DefaultOutboundChannels) Set(name string, channel OutboundChannel) {
	channels.values.Store(name, channel)
	return
}
