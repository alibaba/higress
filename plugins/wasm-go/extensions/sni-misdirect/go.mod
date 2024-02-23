module wasm_go/higress/plugins/wasm-go/extensions/sni_misdirect

go 1.19

replace (
	github.com/alibaba/higress/plugins/wasm-go => ../..
	github.com/tetratelabs/proxy-wasm-go-sdk => github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20240105034322-9a6ac242c3dd
)

require (
	github.com/alibaba/higress/plugins/wasm-go v1.3.1
	github.com/tetratelabs/proxy-wasm-go-sdk v0.22.0
	github.com/google/uuid v1.3.0 // indirect
	github.com/higress-group/nottinygc v0.0.0-20231101025119-e93c4c2f8520 // indirect
	github.com/magefile/mage v1.14.0 // indirect
	github.com/tidwall/gjson v1.14.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
)
