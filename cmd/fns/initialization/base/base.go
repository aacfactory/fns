package base

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"github.com/aacfactory/fns/cmd/generates/processes"
	"github.com/aacfactory/fns/cmd/generates/writers"
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
	process.Add("mod: writing", writers.Unit(mod))
	// configs
	configs, configsErr := NewConfigFiles(dir)
	if configsErr != nil {
		err = configsErr
		return
	}
	configsUints := make([]processes.Unit, 0, 1)
	for _, config := range configs {
		configsUints = append(configsUints, writers.Unit(config))
	}
	process.Add("configs: writing", configsUints...)
	// hooks
	hooks, hooksErr := NewHooksFile(dir)
	if hooksErr != nil {
		err = hooksErr
		return
	}
	process.Add("hooks: writing", writers.Unit(hooks))
	// internal >>>
	generator, generatorErr := NewInternalGeneratorsFile(dir)
	if generatorErr != nil {
		err = generatorErr
		return
	}
	process.Add("generator: writing", writers.Unit(generator))
	// internal <<<
	// modules
	modules, modulesErr := NewModulesFile(path, dir)
	if modulesErr != nil {
		err = modulesErr
		return
	}
	process.Add("modules: writing", writers.Unit(modules))
	// repositories
	repositories, repositoriesErr := NewRepositoryFile(dir)
	if repositoriesErr != nil {
		err = repositoriesErr
		return
	}
	process.Add("repositories: writing", writers.Unit(repositories))
	// main
	main, mainErr := NewMainFile(path, dir)
	if mainErr != nil {
		err = mainErr
		return
	}
	process.Add("main: writing", writers.Unit(main))
	// start
	results := process.Start(ctx)
	for {
		result, ok := <-results
		if !ok {
			break
		}
		fmt.Println(result)
		if result.Error != nil {
			err = result.Error
			_ = process.Abort(1 * time.Second)
			return
		}
	}
	return
}
