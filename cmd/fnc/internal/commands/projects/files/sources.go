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

package files

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/files"
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/processes"
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/sources"
	"path/filepath"
	"time"
)

func Write(ctx context.Context, path string, dir string) (err error) {
	if files.ExistFile(filepath.Join(dir, "go.mod")) {
		err = errors.Warning("fnc: go.mod is exist")
		return
	}
	process := processes.New()
	// mod
	mod, modErr := NewModFile(path, dir)
	if modErr != nil {
		err = modErr
		return
	}
	process.Add("mod: writing", sources.Unit(mod))
	// configs
	configs, configsErr := NewConfigFiles(dir)
	if configsErr != nil {
		err = configsErr
		return
	}
	configsUints := make([]processes.Unit, 0, 1)
	for _, config := range configs {
		configsUints = append(configsUints, sources.Unit(config))
	}
	process.Add("configs: writing", configsUints...)
	// hooks
	hooks, hooksErr := NewHooksFile(dir)
	if hooksErr != nil {
		err = hooksErr
		return
	}
	process.Add("hooks: writing", sources.Unit(hooks))
	// repositories
	repositories, repositoriesErr := NewRepositoryFile(dir)
	if repositoriesErr != nil {
		err = repositoriesErr
		return
	}
	process.Add("repositories: writing", sources.Unit(repositories))
	// modules
	modules, modulesErr := NewModulesFile(path, dir)
	if modulesErr != nil {
		err = modulesErr
		return
	}
	process.Add("modules: writing", sources.Unit(modules))
	// main
	main, mainErr := NewMainFile(path, dir)
	if mainErr != nil {
		err = mainErr
		return
	}
	process.Add("main: writing", sources.Unit(main))
	results := process.Start(ctx)
	for {
		result, ok := <-results
		if !ok {
			break
		}
		fmt.Println(result)
		if result.Error != nil {
			fmt.Println(fmt.Sprintf("%+v", result.Error))
			_ = process.Abort(1 * time.Second)
			break
		}
	}
	return
}
