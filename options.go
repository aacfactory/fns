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

package fns

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/go-playground/validator/v10"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultVersion = "x.y.z"
)

// +-------------------------------------------------------------------------------------------------------------------+

type Option func(*Options) error

var (
	defaultOptions = &Options{
		ConfigRetrieverOption: defaultConfigRetrieverOption(),
		Version:               defaultVersion,
		SecretKey:             secretKey,
		MinPROCS:              1,
		MaxPROCS:              0,
	}
	secretKey = []byte("+-fns")
)

type Options struct {
	ConfigRetrieverOption configuares.RetrieverOption
	Validate              *validator.Validate
	Hooks                 []Hook
	Version               string
	SecretKey             []byte
	MinPROCS              int
	MaxPROCS              int
}

// +-------------------------------------------------------------------------------------------------------------------+

func ConfigRetriever(path string, format string, active string, prefix string, splitter byte) Option {
	return func(o *Options) error {
		path = strings.TrimSpace(path)
		if path == "" {
			return fmt.Errorf("path is empty")
		}
		active = strings.TrimSpace(active)
		format = strings.ToUpper(strings.TrimSpace(format))
		store := configuares.NewFileStore(path, prefix, splitter)
		o.ConfigRetrieverOption = configuares.RetrieverOption{
			Active: active,
			Format: format,
			Store:  store,
		}
		return nil
	}
}

func ConfigActiveFromENV(key string) (active string) {
	v, has := os.LookupEnv(key)
	if !has {
		return
	}
	active = strings.ToLower(strings.TrimSpace(v))
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func Hooks(hooks ...Hook) Option {
	return func(o *Options) error {
		if hooks == nil || len(hooks) == 0 {
			return fmt.Errorf("hooks is empty")
		}
		copy(o.Hooks, hooks)
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func SecretKeyFile(path string) Option {
	return func(options *Options) error {
		path = strings.TrimSpace(path)
		if path == "" {
			return fmt.Errorf("set secret key failed for empty path")
		}
		p, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("set secret key failed for get absolute representation of path failed")
		}
		data, readErr := ioutil.ReadFile(p)
		if readErr != nil {
			return fmt.Errorf("set secret key failed for read file failed, %v", readErr)
		}
		options.SecretKey = bytes.TrimSpace(data)
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func Version(version string) Option {
	return func(options *Options) error {
		version = strings.TrimSpace(version)
		if version == "" {
			return fmt.Errorf("set version failed for empty")
		}
		options.Version = version
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func CustomizeValidate(validate *validator.Validate) Option {
	return func(options *Options) error {
		if validate == nil {
			return fmt.Errorf("set validate failed for nil")
		}
		options.Validate = validate
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func GOPROCS(min int, max int) Option {
	return func(options *Options) error {
		if min < 1 {
			min = 1
		}
		if max < 1 {
			max = 0
		}
		options.MinPROCS = min
		options.MaxPROCS = max
		return nil
	}
}
