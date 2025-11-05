package wasmplugin

import (
	"io"

	"github.com/corazawaf/coraza/v3/debuglog"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

type logger struct {
	debuglog.Logger
}

var _ debuglog.Logger = logger{}

var logPrinterFactory = func(io.Writer) debuglog.Printer {
	return func(lvl debuglog.Level, message, fields string) {
		switch lvl {
		case debuglog.LevelTrace:
			proxywasm.LogTracef("%s %s", message, fields)
		case debuglog.LevelDebug:
			proxywasm.LogDebugf("%s %s", message, fields)
		case debuglog.LevelInfo:
			proxywasm.LogInfof("%s %s", message, fields)
		case debuglog.LevelWarn:
			proxywasm.LogWarnf("%s %s", message, fields)
		case debuglog.LevelError:
			proxywasm.LogErrorf("%s %s", message, fields)
		default:
		}
	}
}

func DefaultLogger() debuglog.Logger {
	return logger{
		debuglog.DefaultWithPrinterFactory(logPrinterFactory),
	}
}

func (l logger) WithLevel(lvl debuglog.Level) debuglog.Logger {
	return logger{l.Logger.WithLevel(lvl)}
}

func (l logger) WithOutput(_ io.Writer) debuglog.Logger {
	proxywasm.LogWarn("Ignoring SecDebugLog directive, debug logs are always routed to proxy logs")
	return l
}
