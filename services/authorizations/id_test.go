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

package authorizations_test

import (
	"github.com/aacfactory/fns/services/authorizations"
	"github.com/aacfactory/json"
	"testing"
)

func TestId_MarshalJSON(t *testing.T) {
	id := authorizations.StringId([]byte("xxx"))
	p, _ := id.MarshalJSON()
	t.Log(string(p))
	nid := authorizations.Id{}
	_ = json.Unmarshal(p, &nid)
	t.Log(nid.String())
}
