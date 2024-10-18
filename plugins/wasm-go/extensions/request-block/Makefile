.DEFAULT:
build:
	tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer' ./main.go
	mv main.wasm ../../../../docker-compose-test/