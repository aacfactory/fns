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

package logs

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"strings"
	"time"
)

type WriterConfig struct {
	Name    string          `json:"name,omitempty" yaml:"name,omitempty"`
	Options json.RawMessage `json:"options,omitempty" yaml:"options,omitempty"`
}

func (writer WriterConfig) Config() (v configures.Config, err error) {
	if len(writer.Options) < 2 {
		writer.Options = []byte{'{', '}'}
	}
	v, err = configures.NewJsonConfig(writer.Options)
	return
}

const (
	TextConsoleFormatter         = ConsoleFormatter("text")
	TextColorfulConsoleFormatter = ConsoleFormatter("text_colorful")
	JsonConsoleFormatter         = ConsoleFormatter("json")
)

type ConsoleFormatter string

func (formatter ConsoleFormatter) Code() logs.ConsoleWriterFormatter {
	switch formatter {
	case TextColorfulConsoleFormatter:
		return logs.ColorTextFormatter
	case JsonConsoleFormatter:
		return logs.JsonFormatter
	default:
		return logs.TextFormatter
	}
}

const (
	Stdout = ConsoleWriterOutType("stdout")
	Stderr = ConsoleWriterOutType("stderr")
	Stdmix = ConsoleWriterOutType("stdout_stderr")
)

type ConsoleWriterOutType string

func (ot ConsoleWriterOutType) Code() logs.ConsoleWriterOutType {
	switch ot {
	case Stdout:
		return logs.StdErr
	case Stderr:
		return logs.StdErr
	default:
		return logs.StdMix
	}
}

const (
	Debug = Level("debug")
	Info  = Level("info")
	Warn  = Level("warn")
	Error = Level("error")
)

type Level string

func (level Level) Code() logs.Level {
	switch level {
	case Debug:
		return logs.DebugLevel
	case Warn:
		return logs.WarnLevel
	case Error:
		return logs.ErrorLevel
	default:
		return logs.InfoLevel
	}
}

type Config struct {
	Level           Level                `json:"level,omitempty" yaml:"level,omitempty"`
	Formatter       ConsoleFormatter     `json:"formatter,omitempty" yaml:"formatter,omitempty"`
	Console         ConsoleWriterOutType `json:"console,omitempty" yaml:"console,omitempty"`
	DisableConsole  bool                 `json:"disableConsole,omitempty" yaml:"disableConsole,omitempty"`
	Consumes        int                  `json:"consumes,omitempty" yaml:"consumes,omitempty"`
	Buffer          int                  `json:"buffer,omitempty" yaml:"buffer,omitempty"`
	SendTimeout     string               `json:"sendTimeout,omitempty" yaml:"sendTimeout,omitempty"`
	ShutdownTimeout string               `json:"shutdownTimeout,omitempty" yaml:"shutdownTimeout,omitempty"`
	Writers         []WriterConfig       `json:"writers,omitempty" yaml:"writer,omitempty"`
}

func (config *Config) GetWriter(name string) (writer configures.Config, err error) {
	for _, writerConfig := range config.Writers {
		if writerConfig.Name == name {
			writer, err = writerConfig.Config()
			return
		}
	}
	writer, err = configures.NewJsonConfig([]byte("{}"))
	return
}

func New(config Config, writers []Writer) (v Logger, err error) {
	options := make([]logs.Option, 0, 1)
	options = append(options, logs.WithLevel(config.Level.Code()))

	if config.DisableConsole {
		options = append(options, logs.DisableConsoleWriter())
	} else {
		options = append(options, logs.WithConsoleWriterOutType(config.Console.Code()))
		options = append(options, logs.WithConsoleWriterFormatter(config.Formatter.Code()))
	}
	if consumes := config.Consumes; consumes > 0 {
		options = append(options, logs.WithConsumes(consumes))
	}
	if buffer := config.Buffer; buffer > 0 {
		options = append(options, logs.WithBuffer(buffer))
	}
	if sendTimeout := strings.TrimSpace(config.SendTimeout); sendTimeout != "" {
		sendTimeouts, parseErr := time.ParseDuration(sendTimeout)
		if parseErr != nil {
			err = errors.Warning("fns: new log failed").WithCause(parseErr).WithMeta("config", "sendTimeout")
			return
		}
		options = append(options, logs.WithSendTimeout(sendTimeouts))
	}
	if shutdownTimeout := strings.TrimSpace(config.ShutdownTimeout); shutdownTimeout != "" {
		shutdownTimeout, parseErr := time.ParseDuration(shutdownTimeout)
		if parseErr != nil {
			err = errors.Warning("fns: new log failed").WithCause(parseErr).WithMeta("config", "shutdownTimeout")
			return
		}
		options = append(options, logs.WithShutdownTimeout(shutdownTimeout))
	}
	if len(writers) > 0 {
		for _, writer := range writers {
			writerConfig, writerConfigErr := config.GetWriter(writer.Name())
			if writerConfigErr != nil {
				err = errors.Warning("fns: new log failed").WithCause(writerConfigErr).WithMeta("writer", writer.Name())
				return
			}
			writerErr := writer.Construct(WriterOptions{
				Config: writerConfig,
			})
			if writerErr != nil {
				err = errors.Warning("fns: new log failed").WithCause(writerErr).WithMeta("writer", writer.Name())
				return
			}
			options = append(options, logs.WithWriter(writer))
		}
	}
	logger, newErr := logs.New(options...)
	if newErr != nil {
		err = errors.Warning("fns: new log failed").WithCause(newErr)
		return
	}
	v = logger
	return
}

type Logger interface {
	logs.Logger
}
