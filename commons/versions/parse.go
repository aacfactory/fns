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

package versions

import (
	"strings"
)

func ParseRange(s string) (left Version, right Version, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		left = Min()
		right = Max()
		return
	}
	versionRange := strings.Split(s, ",")
	leftVersionValue := strings.TrimSpace(versionRange[0])
	if leftVersionValue != "" {
		left, err = Parse(leftVersionValue)
		if err != nil {
			return
		}
	} else {
		left = Min()
	}
	if len(versionRange) > 1 {
		rightVersionValue := strings.TrimSpace(versionRange[1])
		if rightVersionValue != "" {
			right, err = Parse(rightVersionValue)
			if err != nil {
				return
			}
		} else {
			right = Max()
		}
	} else {
		right = Max()
	}
	return
}
