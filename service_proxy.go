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

package fns

type localedServiceProxy struct {
	service Service
}

func (proxy *localedServiceProxy) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = syncResult()
	v, err := proxy.service.Handle(WithNamespace(ctx, proxy.service.Namespace()), fn, argument)
	if err != nil {
		result.Failed(err)
		return
	}
	result.Succeed(v)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type RemotedServiceProxy struct {
	namespace string
	address   string
}

func (proxy *RemotedServiceProxy) Request(ctx Context, fn string, argument Argument) (result Result) {
	// todo: fast http client
	return
}
