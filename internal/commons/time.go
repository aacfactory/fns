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

package commons

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
	if len(value) == 10 {
		result, err = time.Parse("2006-01-02", value)
		return
	}
	if len(value) == len(time.RFC3339) {
		result, err = time.Parse(time.RFC3339, value)
		return
	}
	if len(value) == len(time.RFC3339Nano) {
		result, err = time.Parse(time.RFC3339Nano, value)
		return
	}
	if len(value) == 19 {
		if strings.IndexByte(value, 'T') > 0 {
			result, err = time.Parse("2006-01-02T15:04:05", value)
		} else {
			result, err = time.Parse("2006-01-02 15:04:05", value)
		}
		return
	}
	if len(value) > 19 && strings.IndexByte(value, 'T') > 0 && strings.IndexByte(value, 'Z') > 0 {
		if strings.IndexByte(value, 'T') > 0 && strings.IndexByte(value, 'Z') > 0 {
			// ISO8601
			result, err = time.Parse("2006-01-02T15:04:05.000Z0700", value)
		} else {
			result, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", value)

		}
		return
	}

	err = fmt.Errorf("parse %s to time failed, layout is not supported", value)

	return
}
