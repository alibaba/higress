module github.com/corazawaf/coraza-proxy-wasm

go 1.19

require (
	github.com/alibaba/higress/plugins/wasm-go v0.0.0
	github.com/corazawaf/coraza-wasilibs v0.0.0-20230408002644-e2e3af21f503
	github.com/corazawaf/coraza/v3 v3.0.0-rc.1.0.20230407165813-a18681b1ec28
	github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20240711023527-ba358c48772f
	github.com/tidwall/gjson v1.14.4
)

require (
	github.com/corazawaf/libinjection-go v0.1.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/higress-group/nottinygc v0.0.0-20231101025119-e93c4c2f8520 // indirect
	github.com/magefile/mage v1.14.0 // indirect
	github.com/petar-dambovaliev/aho-corasick v0.0.0-20211021192214-5ab2d9280aa9 // indirect
	github.com/tetratelabs/wazero v1.6.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/resp v0.1.1 // indirect
	github.com/wasilibs/go-aho-corasick v0.3.0 // indirect
	github.com/wasilibs/go-libinjection v0.2.1 // indirect
	github.com/wasilibs/go-re2 v1.0.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	rsc.io/binaryregexp v0.2.0 // indirect
)

replace github.com/alibaba/higress/plugins/wasm-go => ../..
