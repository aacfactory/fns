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

package services

import "github.com/aacfactory/fns/commons/versions"

type Header struct {
	processId        []byte
	requestId        []byte
	endpointId       []byte
	deviceId         []byte
	deviceIp         []byte
	token            []byte
	acceptedVersions versions.Intervals
	internal         bool
}

func (header Header) ProcessId() []byte {
	return header.processId
}

func (header Header) RequestId() []byte {
	return header.requestId
}

func (header Header) EndpointId() []byte {
	return header.endpointId
}

func (header Header) DeviceId() []byte {
	return header.deviceId
}

func (header Header) DeviceIp() []byte {
	return header.deviceIp
}

func (header Header) Token() []byte {
	return header.token
}

func (header Header) AcceptedVersions() versions.Intervals {
	return header.acceptedVersions
}

func (header Header) Internal() bool {
	return header.internal
}
