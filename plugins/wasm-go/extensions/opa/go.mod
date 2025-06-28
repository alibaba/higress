module github.com/alibaba/higress/plugins/wasm-go/extensions/opa

go 1.24

require (
	github.com/higress-group/wasm-go v1.0.0
	github.com/higress-group/proxy-wasm-go-sdk v1.0.0
	github.com/stretchr/testify v1.8.4
	github.com/tidwall/gjson v1.17.3
)

replace github.com/higress-group/proxy-wasm-go-sdk => github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20250611100342-5654e89a7a80

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.5.0 // indirect

	github.com/magefile/mage v1.14.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
