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

package cors

import "bytes"

var (
	comma = []byte{','}
)

func parseHeaderList(headerList [][]byte) [][]byte {
	out := headerList
	copied := false
	for i, v := range headerList {
		needsSplit := bytes.IndexByte(v, ',') != -1
		if !copied {
			if needsSplit {
				split := bytes.Split(v, comma)
				out = make([][]byte, i, len(headerList)+len(split)-1)
				copy(out, headerList[:i])
				for _, s := range split {
					out = append(out, bytes.TrimSpace(s))
				}
				copied = true
			}
		} else {
			if needsSplit {
				split := bytes.Split(v, comma)
				for _, s := range split {
					out = append(out, bytes.TrimSpace(s))
				}
			} else {
				out = append(out, v)
			}
		}
	}
	return out
}
