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

package logger

import (
	"github.com/aacfactory/logs"
	"os"
	"strings"
)

type Config struct {
	Level     string `json:"level" yaml:"level,omitempty"`
	Formatter string `json:"formatter" yaml:"formatter,omitempty"`
	Color     bool   `json:"color" yaml:"color,omitempty"`
}

func (config *Config) Options(name string) Options {
	return Options{
		Name:      name,
		Level:     config.Level,
		Formatter: config.Formatter,
		Color:     config.Color,
	}
}

type Options struct {
	Name      string
	Level     string
	Formatter string
	Color     bool
}

func NewLog(options Options) (v logs.Logger, err error) {
	if options.Name == "" {
		options.Name = "fns"
	}
	formatter := logs.ConsoleFormatter
	if strings.ToLower(strings.TrimSpace(options.Formatter)) == "json" {
		formatter = logs.JsonFormatter
	}
	level := logs.ErrorLevel
	levelValue := strings.ToLower(strings.TrimSpace(options.Level))
	switch levelValue {
	case "debug":
		level = logs.DebugLevel
	case "info":
		level = logs.InfoLevel
	case "warn":
		level = logs.WarnLevel
	case "error":
		level = logs.ErrorLevel
	default:
		level = logs.InfoLevel
	}
	v, err = logs.New(
		logs.WithFormatter(formatter),
		logs.Name(options.Name),
		logs.WithLevel(level),
		logs.Writer(os.Stdout),
		logs.Color(options.Color),
	)
	return
}
