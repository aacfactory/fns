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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/files"
	"github.com/aacfactory/fns/configs"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/goccy/go-yaml"
	"os"
	"path/filepath"
	"strings"
)

func NewConfigFiles(dir string) (v []*ConfigFile, err error) {
	v = make([]*ConfigFile, 0, 1)
	// root
	root, rootErr := NewConfigFile("", dir)
	if rootErr != nil {
		err = rootErr
		return
	}
	v = append(v, root)
	// local
	local, localErr := NewConfigFile("local", dir)
	if localErr != nil {
		err = localErr
		return
	}
	v = append(v, local)
	// dev
	dev, devErr := NewConfigFile("dev", dir)
	if devErr != nil {
		err = devErr
		return
	}
	v = append(v, dev)
	// test
	test, testErr := NewConfigFile("test", dir)
	if testErr != nil {
		err = testErr
		return
	}
	v = append(v, test)
	// prod
	prod, prodErr := NewConfigFile("prod", dir)
	if prodErr != nil {
		err = prodErr
		return
	}
	v = append(v, prod)
	return
}

func NewConfigFile(kind string, dir string) (cf *ConfigFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fnc: new config file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	name := "fns.yaml"
	if kind != "" {
		kind = strings.TrimSpace(strings.ToLower(kind))
		switch kind {
		case "local":
			name = "fns-local.yaml"
			break
		case "dev":
			name = "fns-dev.yaml"
			break
		case "test":
			name = "fns-test.yaml"
			break
		case "prod":
			name = "fns-prod.yaml"
			break
		default:
			err = errors.Warning("").WithCause(errors.Warning("kind is invalid")).WithMeta("kind", kind)
		}
	}
	dir = filepath.ToSlash(filepath.Join(dir, "configs"))
	filename := filepath.ToSlash(filepath.Join(dir, name))
	cf = &ConfigFile{
		kind:     kind,
		dir:      dir,
		filename: filename,
	}
	return
}

type ConfigFile struct {
	kind     string
	dir      string
	filename string
}

func (cf *ConfigFile) Name() (name string) {
	name = cf.filename
	return
}

func (cf *ConfigFile) Write(_ context.Context) (err error) {
	if !files.ExistFile(cf.dir) {
		mdErr := os.MkdirAll(cf.dir, 0644)
		if mdErr != nil {
			err = errors.Warning("fnc: config file write failed").WithCause(mdErr).WithMeta("dir", cf.dir)
			return
		}
	}
	config := configs.Config{}
	switch cf.kind {
	case "local":
		config.Log = logs.Config{
			Level:     "debug",
			Formatter: "console",
			Color:     true,
		}
		config.Runtime = configs.RuntimeConfig{
			Procs:   configs.ProcsConfig{},
			Workers: configs.WorkersConfig{},
			Shared:  shareds.LocalSharedConfig{},
		}
		break
	case "dev":
		config.Log = logs.Config{
			Level:     "info",
			Formatter: "json",
			Color:     false,
		}
		config.Runtime = configs.RuntimeConfig{
			Procs:   configs.ProcsConfig{Min: 2},
			Workers: configs.WorkersConfig{},
			Shared:  shareds.LocalSharedConfig{},
		}
		break
	case "test":
		config.Log = logs.Config{
			Level:     "warn",
			Formatter: "json",
			Color:     false,
		}
		config.Runtime = configs.RuntimeConfig{
			Procs:   configs.ProcsConfig{Min: 2},
			Workers: configs.WorkersConfig{},
			Shared:  shareds.LocalSharedConfig{},
		}
		break
	case "prod":
		config.Log = logs.Config{
			Level:     "error",
			Formatter: "json",
			Color:     false,
		}
		config.Runtime = configs.RuntimeConfig{
			Procs:   configs.ProcsConfig{Min: 8},
			Workers: configs.WorkersConfig{},
			Shared:  shareds.LocalSharedConfig{},
		}
		break
	default:
		config.Transport = transports.Config{
			Port: 18080,
		}
		break
	}
	p, encodeErr := yaml.Marshal(config)
	if encodeErr != nil {
		err = errors.Warning("fnc: config file write failed").WithCause(encodeErr).WithMeta("filename", cf.filename)
		return
	}
	writeErr := os.WriteFile(cf.filename, p, 0644)
	if writeErr != nil {
		err = errors.Warning("fnc: config file write failed").WithCause(writeErr).WithMeta("filename", cf.filename)
		return
	}
	return
}
