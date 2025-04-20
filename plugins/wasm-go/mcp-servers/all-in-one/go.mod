module all-in-one

go 1.24.1

replace quark-search => ../quark-search

replace amap-tools => ../amap-tools

replace github.com/alibaba/higress/plugins/wasm-go => ../..

require (
	amap-tools v0.0.0-00010101000000-000000000000
	github.com/alibaba/higress/plugins/wasm-go v1.4.4-0.20250417132640-fb2e8d5157ad
	quark-search v0.0.0-00010101000000-000000000000
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/higress-group/gjson_template v0.0.0-20250413075336-4c4161ed428b // indirect
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20250402062734-d50d98c305f0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
