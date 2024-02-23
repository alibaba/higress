module jwt-auth

go 1.19

replace (
	github.com/alibaba/higress/plugins/wasm-go => ../..
	github.com/tetratelabs/proxy-wasm-go-sdk => github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20240105034322-9a6ac242c3dd
)

require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0-20230811015533-49269b43032f
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/tetratelabs/proxy-wasm-go-sdk v0.22.0
	github.com/tidwall/gjson v1.16.0
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/magefile/mage v1.14.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/wasilibs/nottinygc v0.3.0 // indirect
)
