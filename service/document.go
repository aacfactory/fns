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

import (
	"github.com/aacfactory/fns/internal/oas"
)

type Document interface {
	Name() (name string)
	Description() (description string)
	Fns() []FnDocument
}

type FnDocument interface {
	Name() (name string)
	Title() (title string)
	Description() (description string)
	Authorization() (has bool)
	Deprecated() (deprecated bool)
	Argument() (argument ElementDocument)
	Result() (result ElementDocument)
}

type ElementDocument interface {
	Key() (v string)
	Schema() (schema *oas.Schema)
}
