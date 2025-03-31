module all-in-one

go 1.24.1

replace quark-search => ../quark-search

replace amap-tools => ../amap-tools

require (
	amap-tools v0.0.0-00010101000000-000000000000
	github.com/alibaba/higress/plugins/wasm-go v1.4.4-0.20250329145934-61b36a20cd9c
	quark-search v0.0.0-00010101000000-000000000000
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20250323151219-d75620c61711 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/tidwall/gjson v1.17.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
