module ai-load-balancer

go 1.24

toolchain go1.24.3

replace github.com/alibaba/higress/plugins/wasm-go => ../..

require (
	github.com/alibaba/higress/plugins/wasm-go v1.4.2
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20250611100342-5654e89a7a80
	github.com/prometheus/client_model v0.6.2
	github.com/tidwall/gjson v1.18.0
	go.uber.org/multierr v1.11.0
)

require (
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/prometheus/common v0.64.0
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
