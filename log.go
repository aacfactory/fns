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
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
	"time"
)

type LogConfig struct {
	Formatter        string `json:"formatter,omitempty"`
	Level            string `json:"level,omitempty"`
	Colorable        bool   `json:"colorable,omitempty"`
	EnableStacktrace bool   `json:"enableStacktrace,omitempty"`
}

func newLogs(name string, config LogConfig) (log Logs) {

	formatter := strings.ToLower(strings.TrimSpace(config.Formatter))
	logFMT := LogConsoleFormatter
	if formatter == "json" {
		logFMT = LogJsonFormatter
	}

	level := strings.ToLower(strings.TrimSpace(config.Level))
	logLEVEL := LogInfoLevel
	if level == "debug" {
		logLEVEL = LogDebugLevel
	} else if level == "info" {
		logLEVEL = LogInfoLevel
	} else if level == "warn" {
		logLEVEL = LogWarnLevel
	} else if level == "error" {
		logLEVEL = LogErrorLevel
	}

	opt := logOption{
		Name:             name,
		Formatter:        logFMT,
		ActiveLevel:      logLEVEL,
		Colorable:        config.Colorable,
		EnableStacktrace: config.EnableStacktrace,
	}

	log = createLog(opt)

	return
}

type Logs interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Sync() error
}

// +-------------------------------------------------------------------------------------------------------------------+

type LogField struct {
	Key   string
	Value interface{}
}

func LogF(key string, value interface{}) LogField {
	return LogField{
		Key:   key,
		Value: value,
	}
}

func LogErrorField(err error) LogField {
	return LogField{
		Key:   "error",
		Value: err,
	}
}

func newCodeErrorMarshalLogObject(err CodeError) codeErrorMarshalLogObject {
	return codeErrorMarshalLogObject{
		err: err,
	}
}

type codeErrorMarshalLogObject struct {
	err CodeError
}

func (o codeErrorMarshalLogObject) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	err := o.err
	fn, file, line := err.Stacktrace()

	enc.AddString("id", err.Id())
	enc.AddString("code", err.Code())
	enc.AddInt("failureCode", err.FailureCode())
	if err.Meta() != nil && len(err.Meta()) > 0 {
		meta := codeErrorMetaMarshalLogObject{
			meta: err.Meta(),
		}
		_ = enc.AddObject("meta", meta)
	}
	enc.AddString("message", err.Message())
	_ = enc.AddObject("stacktrace", codeErrorStacktraceMarshalLogObject{
		fn:   fn,
		file: file,
		line: line,
	})
	return nil
}

type codeErrorStacktraceMarshalLogObject struct {
	fn   string
	file string
	line int
}

func (o codeErrorStacktraceMarshalLogObject) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("fn", o.fn)
	enc.AddString("file", o.file)
	enc.AddInt("line", o.line)
	return nil
}

type codeErrorMetaMarshalLogObject struct {
	meta map[string][]string
}

func (o codeErrorMetaMarshalLogObject) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for key, values := range o.meta {
		enc.AddString(key, strings.Join(values, ","))
	}
	return nil
}

// +-------------------------------------------------------------------------------------------------------------------+

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	s := t.Format(time.RFC3339Nano)
	s1 := s[:strings.LastIndexByte(s, '.')]
	s2 := s[strings.LastIndexByte(s, '.')+1 : strings.LastIndexByte(s, '+')]
	for i := len(s2); i < 9; i++ {
		s2 = s2 + "0"
	}
	s3 := s[strings.LastIndexByte(s, '+')+1:]
	enc.AppendString(fmt.Sprintf("%s.%s+%s", s1, s2, s3))
}

// +-------------------------------------------------------------------------------------------------------------------+

// Foreground colors.
const (
	red     color = 35
	yellow  color = 33
	blue    color = 36
	magenta color = 37
)

// color represents a text color.
type color uint8

// Add adds the coloring to the given string.
func (c color) Add(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), s)
}

// +-------------------------------------------------------------------------------------------------------------------+

type LogLevel string

func zapLogLevel(name LogLevel) zapcore.Level {
	value := strings.ToLower(string(name))
	switch value {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

var (
	_zapLoglevelToColor = map[zapcore.Level]color{
		zapcore.DebugLevel:  magenta,
		zapcore.InfoLevel:   blue,
		zapcore.WarnLevel:   yellow,
		zapcore.ErrorLevel:  red,
		zapcore.DPanicLevel: red,
		zapcore.PanicLevel:  red,
		zapcore.FatalLevel:  red,
	}
)

func zapLogCapitalColorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	s := ""
	color, ok := _zapLoglevelToColor[l]
	if ok {
		s = color.Add(l.CapitalString())
	}
	enc.AppendString(s)
}

func zapLogFullCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(TrimRCFilepath(caller.FullPath()))
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	LogConsoleFormatter = LogFormatter("console")
	LogJsonFormatter    = LogFormatter("json")

	LogDebugLevel = LogLevel("debug")
	LogInfoLevel  = LogLevel("info")
	LogWarnLevel  = LogLevel("warn")
	LogErrorLevel = LogLevel("error")

	defaultLogName = "FNS"
)

type LogFormatter string

type logOption struct {
	Name             string       `json:"name,omitempty"`
	Formatter        LogFormatter `json:"formatter,omitempty"`
	ActiveLevel      LogLevel     `json:"activeLevel,omitempty"`
	Colorable        bool         `json:"colorable,omitempty"`
	EnableCaller     bool         `json:"enableCaller,omitempty"`
	EnableStacktrace bool         `json:"enableStacktrace,omitempty"`
}

func createLog(option logOption) Logs {

	name := strings.TrimSpace(option.Name)
	if name == "" {
		name = defaultLogName
	}

	formatter := option.Formatter
	if formatter == "" {
		formatter = LogConsoleFormatter
	}
	activeLevel := option.ActiveLevel
	if activeLevel == "" {
		activeLevel = LogInfoLevel
	}

	zapLevel := zapLogLevel(activeLevel)
	var callerEncoder zapcore.CallerEncoder
	if option.EnableCaller {
		if formatter == LogJsonFormatter {
			callerEncoder = zapLogFullCallerEncoder
			option.Colorable = false
		} else {
			callerEncoder = zapLogFullCallerEncoder
		}
	}

	var encodeLevel zapcore.LevelEncoder
	if !option.Colorable {
		encodeLevel = zapcore.CapitalLevelEncoder
	} else {
		encodeLevel = zapLogCapitalColorLevelEncoder
	}
	encodingConfig := zapcore.EncoderConfig{
		TimeKey:        "_T",
		LevelKey:       "_L",
		NameKey:        "_N",
		CallerKey:      "_C",
		MessageKey:     "_M",
		StacktraceKey:  "_S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel,
		EncodeTime:     timeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   callerEncoder,
	}

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLevel),
		Development:       false,
		Encoding:          string(formatter),
		EncoderConfig:     encodingConfig,
		DisableStacktrace: !option.EnableStacktrace,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		InitialFields:     map[string]interface{}{"app": name},
	}

	log, createEr := config.Build()
	if createEr != nil {
		panic(fmt.Errorf("logs create failed, %v", createEr))
	}

	return log.Sugar()
}

func LogWith(log Logs, fields ...LogField) (_log Logs) {
	sLog, ok := log.(*zap.SugaredLogger)
	if !ok {
		panic(fmt.Errorf("logs with fields failed, it is not *zap.SugaredLogger"))
		return
	}

	if fields == nil || len(fields) == 0 {
		_log = sLog.Desugar().Sugar()
		return
	}

	kvs := make([]zap.Field, 0, 1)
	for _, field := range fields {
		if field.Key == "error" {
			codeErr, isCodeErr := field.Value.(CodeError)
			if isCodeErr {
				kvs = append(kvs, zap.Object(field.Key, newCodeErrorMarshalLogObject(codeErr)))
				continue
			}
		} else {
			kvs = append(kvs, zap.Any(field.Key, field.Value))
		}
	}

	_log = sLog.Desugar().With(kvs...).Sugar()
	return
}

func LogWithCodeError(log Logs, err CodeError) (_log Logs) {
	sLog, ok := log.(*zap.SugaredLogger)
	if !ok {
		panic(fmt.Errorf("logs with fields failed, it is not *zap.SugaredLogger"))
		return
	}
	_log = sLog.Desugar().With(zap.Object("error", newCodeErrorMarshalLogObject(err))).Sugar()
	return
}
