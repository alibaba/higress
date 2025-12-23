.DEFAULT:
build:
	tinygo build -o ai-proxy.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100' ./main.go
	mv ai-proxy.wasm ../../../../docker-compose-test/

build-go:
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm main.go