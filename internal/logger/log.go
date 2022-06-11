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
	"fmt"
	"github.com/aacfactory/logs"
	"os"
	"strings"
)

type Printf struct {
	core logs.Logger
}

func (p *Printf) Printf(layout string, v ...interface{}) {
	if p.core.DebugEnabled() {
		p.core.Debug().Message(fmt.Sprintf("fns: %s", fmt.Sprintf(layout, v...)))
	}
}

type LogOptions struct {
	Name      string
	Level     string `copy:"Level"`
	Formatter string `copy:"Formatter"`
	Color     bool   `copy:"Color"`
}

func NewLog(options LogOptions) (v logs.Logger, err error) {
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
