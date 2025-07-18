.PHONY: reload

build:
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ./main.wasm ./main.go

reload:
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ./main.wasm ./main.go
	./envoy -c envoy.yaml --concurrency 0 --log-level info --component-log-level wasm:debug

start:
	./envoy -c envoy.yaml --concurrency 0 --log-level info --component-log-level wasm:debug