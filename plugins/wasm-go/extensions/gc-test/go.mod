module github.com/alibaba/higress/plugins/wasm-go/extensions/gc-test

go 1.24

replace github.com/wasilibs/nottinygc v0.5.1 => github.com/higress-group/nottinygc v0.0.0-20231019105920-c4d985d443e1

require (
	github.com/higress-group/wasm-go v1.0.0
	github.com/higress-group/proxy-wasm-go-sdk v1.0.0
	github.com/tidwall/gjson v1.17.3
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/higress-group/nottinygc v0.0.0-20231101025119-e93c4c2f8520 // indirect
	github.com/magefile/mage v1.14.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
)
