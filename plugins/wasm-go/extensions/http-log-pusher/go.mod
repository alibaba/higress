module github.com/alibaba/higress/plugins/wasm-go/extensions/http-log-pusher

go 1.24.1

toolchain go1.24.4

require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20250822030947-8345453fddd0
	github.com/tidwall/gjson v1.18.0
	github.com/stretchr/testify v1.9.0
)

replace github.com/alibaba/higress/plugins/wasm-go => ../..

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
