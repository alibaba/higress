module github.com/alibaba/higress/plugins/wasm-go/extensions/gc-test

go 1.19

replace github.com/alibaba/higress/plugins/wasm-go => ../..

replace github.com/wasilibs/nottinygc v0.5.1 => github.com/higress-group/nottinygc v0.0.0-20231019105920-c4d985d443e1

require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20240711023527-ba358c48772f
	github.com/tidwall/gjson v1.14.3
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/higress-group/nottinygc v0.0.0-20231101025119-e93c4c2f8520 // indirect
	github.com/magefile/mage v1.14.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
)
