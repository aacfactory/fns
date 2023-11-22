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

package wildcard

import (
	"bytes"
)

func Match(pattern []byte, target []byte) (ok bool) {
	ok = New(pattern).Match(target)
	return
}

func New(pattern []byte) (w *Wildcard) {
	if len(pattern) == 1 && pattern[0] == '*' {
		w = &Wildcard{
			prefix: nil,
			suffix: nil,
		}
		return
	}
	idx := bytes.IndexByte(pattern, '*')
	if idx < 0 {
		w = &Wildcard{
			prefix: pattern,
			suffix: nil,
		}
		return
	}
	w = &Wildcard{
		prefix: pattern[0:idx],
		suffix: pattern[idx+1:],
	}
	return
}

type Wildcard struct {
	prefix []byte
	suffix []byte
}

func (w *Wildcard) Match(s []byte) bool {
	if len(w.suffix) == 0 {
		return bytes.Equal(w.prefix, s)
	}
	return len(s) >= len(w.prefix)+len(w.suffix) && bytes.HasPrefix(s, w.prefix) && bytes.HasSuffix(s, w.suffix)
}
