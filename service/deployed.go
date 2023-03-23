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

package service

func newDeployed() *deployed {
	return &deployed{
		chs: make([]chan map[string]*endpoint, 0, 1),
	}
}

type deployed struct {
	chs []chan map[string]*endpoint
}

func (d *deployed) acquire() (ch <-chan map[string]*endpoint) {
	chf := make(chan map[string]*endpoint, 1)
	d.chs = append(d.chs, chf)
	ch = chf
	return
}

func (d *deployed) publish(v map[string]*endpoint) {
	for _, ch := range d.chs {
		ch <- v
		close(ch)
	}
}
