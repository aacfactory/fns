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

package configs

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	activeSystemEnvKey = "FNS-ACTIVE"
)

func DefaultConfigRetrieverOption() (option configures.RetrieverOption) {
	path, pathErr := filepath.Abs("./configs")
	if pathErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: create default config retriever failed, cant not get absolute representation of './configs'").WithCause(pathErr)))
		return
	}
	active, _ := os.LookupEnv(activeSystemEnvKey)
	active = strings.TrimSpace(active)
	store := configures.NewFileStore(path, "fns", '-')
	option = configures.RetrieverOption{
		Active: active,
		Format: "YAML",
		Store:  store,
	}
	return
}
