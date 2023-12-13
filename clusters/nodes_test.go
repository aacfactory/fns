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

package clusters_test

import (
	"github.com/aacfactory/fns/clusters"
	"testing"
)

func TestNodes_Difference(t *testing.T) {
	s1 := clusters.Nodes{
		{
			Id: "11",
		},
		{
			Id: "12",
		},
	}
	s2 := clusters.Nodes{
		{
			Id: "11",
		},
		{
			Id: "12",
		},
	}
	events := s2.Difference(s1)
	for _, event := range events {
		t.Log(event.Kind.String(), event.Node.Id)
	}
}
