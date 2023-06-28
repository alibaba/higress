package main

import (
	"github.com/corazawaf/coraza-proxy-wasm/wasmplugin"
	wasilibs "github.com/corazawaf/coraza-wasilibs"
)

func main() {
	wasilibs.RegisterRX()
	wasilibs.RegisterPM()
	wasilibs.RegisterSQLi()
	wasilibs.RegisterXSS()
	wasmplugin.PluginStart()
}
