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

package tracings

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/logs"
)

type ReporterOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Reporter interface {
	Construct(options ReporterOptions) (err error)
	Report(ctx context.Context, trace *Trace)
}
