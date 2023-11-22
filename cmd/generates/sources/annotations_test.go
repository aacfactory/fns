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

package sources_test

import (
	"fmt"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"testing"
)

func TestParseAnnotations(t *testing.T) {
	s := `@title title
@desc >>>

	name1:
	zh: chinese
	en: english
	name2:
	zh: chinese
	en: english

<<<
@barrier
@auth
@permission
@sql:tx name
@cache get set`
	annos, err := sources.ParseAnnotations(s)
	if err != nil {
		fmt.Println(fmt.Sprintf("%+v", err))
		return
	}
	for _, anno := range annos {
		fmt.Println(anno.Name, len(anno.Params), fmt.Sprintf("%+v", anno.Params))
	}
}
