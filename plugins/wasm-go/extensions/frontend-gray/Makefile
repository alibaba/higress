.PHONY: reload

build:
	tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer' ./main.go

reload:
	tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer' ./main.go
	./envoy -c envoy.yaml --concurrency 0 --log-level info --component-log-level wasm:debug

start:
	./envoy -c envoy.yaml --concurrency 0 --log-level info --component-log-level wasm:debug