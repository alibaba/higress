// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wrapper

import (
	"fmt"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
)

type LogLevel uint32

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelCritical
)

type LogWrapper struct {
	pluginName string
}

func (l LogWrapper) log(level LogLevel, msg string) {
	msg = fmt.Sprintf("[%s] %s", l.pluginName, msg)
	switch level {
	case LogLevelTrace:
		proxywasm.LogTrace(msg)
	case LogLevelDebug:
		proxywasm.LogDebug(msg)
	case LogLevelInfo:
		proxywasm.LogInfo(msg)
	case LogLevelWarn:
		proxywasm.LogWarn(msg)
	case LogLevelError:
		proxywasm.LogError(msg)
	case LogLevelCritical:
		proxywasm.LogCritical(msg)
	}
}

func (l LogWrapper) logFormat(level LogLevel, format string, args ...interface{}) {
	format = fmt.Sprintf("[%s] %s", l.pluginName, format)
	switch level {
	case LogLevelTrace:
		proxywasm.LogTracef(format, args...)
	case LogLevelDebug:
		proxywasm.LogDebugf(format, args...)
	case LogLevelInfo:
		proxywasm.LogInfof(format, args...)
	case LogLevelWarn:
		proxywasm.LogWarnf(format, args...)
	case LogLevelError:
		proxywasm.LogErrorf(format, args...)
	case LogLevelCritical:
		proxywasm.LogCriticalf(format, args...)
	}
}

func (l LogWrapper) Trace(msg string) {
	l.log(LogLevelTrace, msg)
}

func (l LogWrapper) Tracef(format string, args ...interface{}) {
	l.logFormat(LogLevelTrace, format, args...)
}

func (l LogWrapper) Debug(msg string) {
	l.log(LogLevelDebug, msg)
}

func (l LogWrapper) Debugf(format string, args ...interface{}) {
	l.logFormat(LogLevelDebug, format, args...)
}

func (l LogWrapper) Info(msg string) {
	l.log(LogLevelInfo, msg)
}

func (l LogWrapper) Infof(format string, args ...interface{}) {
	l.logFormat(LogLevelInfo, format, args...)
}

func (l LogWrapper) Warn(msg string) {
	l.log(LogLevelWarn, msg)
}

func (l LogWrapper) Warnf(format string, args ...interface{}) {
	l.logFormat(LogLevelWarn, format, args...)
}

func (l LogWrapper) Error(msg string) {
	l.log(LogLevelError, msg)
}

func (l LogWrapper) Errorf(format string, args ...interface{}) {
	l.logFormat(LogLevelError, format, args...)
}

func (l LogWrapper) Critical(msg string) {
	l.log(LogLevelCritical, msg)
}

func (l LogWrapper) Criticalf(format string, args ...interface{}) {
	l.logFormat(LogLevelCritical, format, args...)
}
