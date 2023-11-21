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

package logs

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"os"
	"strings"
)

type WriterConfig struct {
	Name    string          `json:"name" yaml:"name"`
	Options json.RawMessage `json:"options" yaml:"options"`
}

type Config struct {
	Level     string       `json:"level" yaml:"level,omitempty"`
	Formatter string       `json:"formatter" yaml:"formatter,omitempty"`
	Color     bool         `json:"color" yaml:"color,omitempty"`
	Writer    WriterConfig `json:"writer" yaml:"writer"`
}

func New(name string, config Config) (v Logger, err error) {
	formatter := logs.ConsoleFormatter
	if strings.ToLower(strings.TrimSpace(config.Formatter)) == "json" {
		formatter = logs.JsonFormatter
	}
	level := logs.ErrorLevel
	levelValue := strings.ToLower(strings.TrimSpace(config.Level))
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
	writer, hasWriter := getWriter(config.Writer.Name)
	if !hasWriter {
		err = errors.Warning("fns: new log failed").WithCause(fmt.Errorf("writer was not found")).WithMeta("writer", config.Writer.Name)
		return
	}
	if len(config.Writer.Options) < 2 {
		config.Writer.Options = []byte{'{', '}'}
	}
	writerConfig, writerConfigErr := configures.NewJsonConfig(config.Writer.Options)
	if writerConfigErr != nil {
		err = errors.Warning("fns: new log failed").WithCause(writerConfigErr).WithMeta("writer", config.Writer.Name)
		return
	}
	writerErr := writer.Construct(WriterOptions{
		Config: writerConfig,
	})
	if writerErr != nil {
		err = errors.Warning("fns: new log failed").WithCause(writerErr).WithMeta("writer", config.Writer.Name)
		return
	}
	core, coreErr := logs.New(
		logs.WithFormatter(formatter),
		logs.Name(name),
		logs.WithLevel(level),
		logs.Writer(os.Stdout),
		logs.Color(config.Color),
		logs.Writer(writer),
	)
	if coreErr != nil {
		writer.Shutdown(context.TODO())
		err = errors.Warning("fns: new log failed").WithCause(coreErr).WithMeta("writer", config.Writer.Name)
		return
	}
	v = &logger{
		Logger: core,
		w:      writer,
	}
	return
}

type Logger interface {
	logs.Logger
	Shutdown(ctx context.Context)
}

type logger struct {
	logs.Logger
	w Writer
}

func (l *logger) Shutdown(ctx context.Context) {
	l.w.Shutdown(ctx)
}
