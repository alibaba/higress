module github.com/alibaba/higress/plugins/wasm-go/extensions/request-block

go 1.18

replace github.com/alibaba/higress/plugins/wasm-go => ../..

require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20240711023527-ba358c48772f
	github.com/tidwall/gjson v1.14.3
	github.com/wasilibs/go-re2 v1.6.0
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/higress-group/nottinygc v0.0.0-20231101025119-e93c4c2f8520 // indirect
	github.com/magefile/mage v1.15.1-0.20230912152418-9f54e0f83e2a // indirect
	github.com/tetratelabs/wazero v1.7.2 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
	golang.org/x/sys v0.21.0 // indirect
)
