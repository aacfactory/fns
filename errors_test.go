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

package fns_test

import (
	"fmt"
	"github.com/aacfactory/fns"
	"testing"
)

func TestNewCodeError(t *testing.T) {
	err := fns.NewCodeError(500, "***FOO***", "bar")
	fmt.Println(err)
	fmt.Println(fns.ServiceError("foo"))
	fmt.Println(fns.InvalidArgumentError("foo"))
	fmt.Println(fns.InvalidArgumentErrorWithDetails("foo"))
	fmt.Println(fns.UnauthorizedError("foo"))
	fmt.Println(fns.ForbiddenError("foo"))
	fmt.Println(fns.ForbiddenErrorWithReason("foo", "role", "bar"))
	fmt.Println(fns.NotFoundError("foo"))
	fmt.Println(fns.NotImplementedError("foo"))
	fmt.Println(fns.UnavailableError("foo"))
}

func TestCodeError_ToJson(t *testing.T) {
	fmt.Println(string(fns.ServiceError("x").ToJson()))
}

func TestFromJson(t *testing.T) {
	err := fns.ServiceError("x")
	v := err.ToJson()
	fmt.Println(fns.DecodeErrorFromJson(v))
}

func TestTransfer(t *testing.T) {
	var err error = fns.ServiceError("x")
	codeErr, ok := fns.TransferError(err)
	fmt.Println(ok, codeErr.String())
}
