.DEFAULT:
build:
	tinygo build -o ai-proxy.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100' ./main.go
	mv ai-proxy.wasm ../../../../docker-compose-test/