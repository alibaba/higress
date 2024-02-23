module wasm-demo

go 1.18

replace github.com/alibaba/higress/plugins/wasm-go => ../..
replace github.com/tetratelabs/proxy-wasm-go-sdk => github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20240105034322-9a6ac242c3dd

require (
	github.com/tetratelabs/proxy-wasm-go-sdk v0.22.0
	github.com/tidwall/gjson v1.14.3
)

require (
	github.com/higress-group/nottinygc v0.0.0-20231101025119-e93c4c2f8520 // indirect
	github.com/magefile/mage v1.14.0 // indirect
)

require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0-20221116034346-4eb91e6918b8
	github.com/google/uuid v1.3.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
)
