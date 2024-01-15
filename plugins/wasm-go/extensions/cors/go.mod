module cors

go 1.19

replace github.com/alibaba/higress/plugins/wasm-go => ../..
replace github.com/tetratelabs/proxy-wasm-go-sdk => github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20240105034322-9a6ac242c3dd

require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0-20230519024024-625c06e58f91
	github.com/stretchr/testify v1.8.3
	github.com/tetratelabs/proxy-wasm-go-sdk v0.22.0
	github.com/tidwall/gjson v1.14.4
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/higress-group/nottinygc v0.0.0-20231101025119-e93c4c2f8520 // indirect
	github.com/magefile/mage v1.14.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
