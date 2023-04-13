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

package sources

import (
	"bufio"
	"bytes"
	"github.com/aacfactory/errors"
	"io"
	"strings"
)

func ParseAnnotations(s string) (annotations Annotations, err error) {
	annotations = make(map[string]string)
	if s == "" || !strings.Contains(s, "@") {
		return
	}
	currentKey := ""
	currentBody := bytes.NewBuffer(make([]byte, 0, 1))
	blockReading := false
	reader := bufio.NewReader(bytes.NewReader([]byte(s)))
	for {
		line, _, readErr := reader.ReadLine()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			err = errors.Warning("sources: parse annotations failed").WithCause(readErr).WithMeta("source", s)
			return
		}
		if line == nil {
			continue
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '@' {
			if blockReading {
				currentBody.Write(line)
				if len(line) > 0 {
					currentBody.WriteByte('\n')
				}
				continue
			}
			if len(line) == 1 {
				continue
			}
			if currentKey != "" {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()
			}
			idx := bytes.IndexByte(line, ' ')
			if idx < 0 {
				currentKey = string(line[1:])
				continue
			}
			currentKey = string(line[1:idx])
			line = bytes.TrimSpace(line[idx:])
		}
		if len(line) == 0 {
			continue
		}
		if blockReading {
			remains, hasBlockEnd := bytes.CutSuffix(line, []byte{'<', '<', '<'})
			currentBody.Write(remains)
			if hasBlockEnd {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()

				blockReading = false
			} else {
				if len(remains) > 0 {
					currentBody.WriteByte('\n')
				}
			}
			continue
		}
		line, blockReading = bytes.CutPrefix(line, []byte{'>', '>', '>'})
		if blockReading && currentKey != "" {
			remains, hasBlockEnd := bytes.CutSuffix(line, []byte{'<', '<', '<'})
			currentBody.Write(remains)
			if hasBlockEnd {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()

				blockReading = false
			} else {
				if len(remains) > 0 {
					currentBody.WriteByte('\n')
				}
			}
			continue
		} else if currentKey != "" {
			currentBody.Write(line)

			annotations.set(currentKey, currentBody.String())
			currentKey = ""
			currentBody.Reset()
		}

	}
	if currentKey != "" {
		annotations.set(currentKey, currentBody.String())
		currentKey = ""
		currentBody.Reset()
	}
	return
}

type Annotations map[string]string

func (annotations Annotations) Get(key string) (value string, has bool) {
	value, has = annotations[key]
	return
}

func (annotations Annotations) set(key string, value string) {
	value, _ = strings.CutSuffix(value, "\n")
	value = strings.ReplaceAll(value, "'>>>'", ">>>")
	value = strings.ReplaceAll(value, "'<<<'", "<<<")
	annotations[key] = value
}
