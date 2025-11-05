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

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
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

type DefaultLog struct {
	pluginName string
	pluginID   string
}

func (l *DefaultLog) log(level LogLevel, msg string) {
	requestIDRaw, _ := proxywasm.GetProperty([]string{"x_request_id"})
	requestID := string(requestIDRaw)
	if requestID == "" {
		requestID = "nil"
	}
	msg = fmt.Sprintf("[%s] [%s] [%s] %s", l.pluginName, l.pluginID, requestID, msg)
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

func (l *DefaultLog) logFormat(level LogLevel, format string, args ...interface{}) {
	requestIDRaw, _ := proxywasm.GetProperty([]string{"x_request_id"})
	requestID := string(requestIDRaw)
	if requestID == "" {
		requestID = "nil"
	}
	format = fmt.Sprintf("[%s] [%s] [%s] %s", l.pluginName, l.pluginID, requestID, format)
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

func (l *DefaultLog) Trace(msg string) {
	l.log(LogLevelTrace, msg)
}

func (l *DefaultLog) Tracef(format string, args ...interface{}) {
	l.logFormat(LogLevelTrace, format, args...)
}

func (l *DefaultLog) Debug(msg string) {
	l.log(LogLevelDebug, msg)
}

func (l *DefaultLog) Debugf(format string, args ...interface{}) {
	l.logFormat(LogLevelDebug, format, args...)
}

func (l *DefaultLog) Info(msg string) {
	l.log(LogLevelInfo, msg)
}

func (l *DefaultLog) Infof(format string, args ...interface{}) {
	l.logFormat(LogLevelInfo, format, args...)
}

func (l *DefaultLog) Warn(msg string) {
	l.log(LogLevelWarn, msg)
}

func (l *DefaultLog) Warnf(format string, args ...interface{}) {
	l.logFormat(LogLevelWarn, format, args...)
}

func (l *DefaultLog) Error(msg string) {
	l.log(LogLevelError, msg)
}

func (l *DefaultLog) Errorf(format string, args ...interface{}) {
	l.logFormat(LogLevelError, format, args...)
}

func (l *DefaultLog) Critical(msg string) {
	l.log(LogLevelCritical, msg)
}

func (l *DefaultLog) Criticalf(format string, args ...interface{}) {
	l.logFormat(LogLevelCritical, format, args...)
}

func (l *DefaultLog) ResetID(pluginID string) {
	l.pluginID = pluginID
}
