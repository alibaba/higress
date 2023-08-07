module wasm-demo

go 1.18

replace github.com/alibaba/higress/plugins/wasm-go => ../..

require (
	github.com/mse-group/wasm-extensions-go v1.0.1
	github.com/tetratelabs/proxy-wasm-go-sdk v0.19.1-0.20220822060051-f9d179a57f8c
	github.com/tidwall/gjson v1.14.3
)

require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0-20221116034346-4eb91e6918b8
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redis/redis v6.15.9+incompatible // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
)
