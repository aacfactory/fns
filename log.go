package fns

import (
	"github.com/aacfactory/logs"
	"strings"
)

type LogConfig struct {
	Formatter        string `json:"formatter,omitempty"`
	Level            string `json:"level,omitempty"`
	Colorable        bool   `json:"colorable,omitempty"`
	EnableStacktrace bool   `json:"enableStacktrace,omitempty"`
}

func newLogs(name string, config LogConfig) (log Logs) {

	formatter := strings.ToLower(strings.TrimSpace(config.Formatter))
	logFMT := logs.LogConsoleFormatter
	if formatter == "json" {
		logFMT = logs.LogJsonFormatter
	}

	level := strings.ToLower(strings.TrimSpace(config.Level))
	logLEVEL := logs.LogInfoLevel
	if level == "debug" {
		logLEVEL = logs.LogDebugLevel
	} else if level == "info" {
		logLEVEL = logs.LogInfoLevel
	} else if level == "warn" {
		logLEVEL = logs.LogWarnLevel
	} else if level == "error" {
		logLEVEL = logs.LogErrorLevel
	}

	opt := logs.LogOption{
		Name:             name,
		Formatter:        logFMT,
		ActiveLevel:      logLEVEL,
		Colorable:        config.Colorable,
		EnableStacktrace: config.EnableStacktrace,
	}

	log = logs.New(opt)

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
