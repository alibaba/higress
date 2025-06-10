module github.com/alibaba/higress/plugins/wasm-go/extensions/test-foreign-function

go 1.24

toolchain go1.24.3

replace github.com/alibaba/higress/plugins/wasm-go => ../..

require (
	github.com/alibaba/higress/plugins/wasm-go v1.3.6-0.20240522012622-fc6a6aad8906
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20250530061616-857d6211121d
	github.com/tidwall/gjson v1.17.3
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
)
