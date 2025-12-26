package pkg

import (
	"github.com/higress-group/wasm-go/pkg/log"
)

func init() {
	// Initialize mock logger for testing
	log.SetPluginLog(&mockLogger{})
}

type mockLogger struct{}

func (m *mockLogger) Trace(msg string)                             {}
func (m *mockLogger) Tracef(format string, args ...interface{})    {}
func (m *mockLogger) Debug(msg string)                             {}
func (m *mockLogger) Debugf(format string, args ...interface{})    {}
func (m *mockLogger) Info(msg string)                              {}
func (m *mockLogger) Infof(format string, args ...interface{})     {}
func (m *mockLogger) Warn(msg string)                              {}
func (m *mockLogger) Warnf(format string, args ...interface{})     {}
func (m *mockLogger) Error(msg string)                             {}
func (m *mockLogger) Errorf(format string, args ...interface{})    {}
func (m *mockLogger) Critical(msg string)                          {}
func (m *mockLogger) Criticalf(format string, args ...interface{}) {}
func (m *mockLogger) ResetID(pluginID string)                      {}
