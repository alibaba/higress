module github.com/alibaba/higress/plugins/wasm-go/extensions/basic-auth

go 1.19


replace github.com/alibaba/higress/plugins/wasm-go => ../..


require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0
	github.com/pkg/errors v0.9.1
	github.com/tetratelabs/proxy-wasm-go-sdk v0.22.0
	github.com/tidwall/gjson v1.14.3
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/magefile/mage v1.14.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/wasilibs/nottinygc v0.3.0 // indirect
)
