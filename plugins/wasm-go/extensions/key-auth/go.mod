module key-auth

go 1.24

require (
	github.com/higress-group/wasm-go v1.0.0
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20240711023527-ba358c48772f
	github.com/tidwall/gjson v1.14.4
)

replace github.com/higress-group/proxy-wasm-go-sdk => github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20250611100342-5654e89a7a80

require (
	github.com/google/uuid v1.3.0 // indirect

	github.com/magefile/mage v1.14.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
)
