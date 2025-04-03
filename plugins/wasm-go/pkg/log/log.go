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

package log

type Log interface {
	Trace(msg string)
	Tracef(format string, args ...interface{})
	Debug(msg string)
	Debugf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
	Critical(msg string)
	Criticalf(format string, args ...interface{})
	ResetID(pluginID string)
}

var pluginLog Log

func SetPluginLog(log Log) {
	pluginLog = log
}

func Trace(msg string) {
	pluginLog.Trace(msg)
}

func Tracef(format string, args ...interface{}) {
	pluginLog.Tracef(format, args...)
}

func Debug(msg string) {
	pluginLog.Debug(msg)
}

func Debugf(format string, args ...interface{}) {
	pluginLog.Debugf(format, args...)
}

func Info(msg string) {
	pluginLog.Info(msg)
}

func Infof(format string, args ...interface{}) {
	pluginLog.Infof(format, args...)
}

func Warn(msg string) {
	pluginLog.Warn(msg)
}

func Warnf(format string, args ...interface{}) {
	pluginLog.Warnf(format, args...)
}

func Error(msg string) {
	pluginLog.Error(msg)
}

func Errorf(format string, args ...interface{}) {
	pluginLog.Errorf(format, args...)
}

func Critical(msg string) {
	pluginLog.Critical(msg)
}

func Criticalf(format string, args ...interface{}) {
	pluginLog.Criticalf(format, args...)
}
