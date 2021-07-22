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

import (
	"fmt"
	"strings"
	"time"
)

func ParseTime(value string) (result time.Time, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		result = time.Time{}
		return
	}
	result, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return
	}

	result, err = time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return
	}

	result, err = time.Parse("2006-01-02T15:04:05", value)
	if err == nil {
		return
	}

	result, err = time.Parse("2006-01-02 15:04:05", value)
	if err == nil {
		return
	}

	// ISO8601
	result, err = time.Parse("2006-01-02T15:04:05.000Z0700", value)
	if err == nil {
		return
	}

	err = fmt.Errorf("parse %s to time failed", value)
	return
}
