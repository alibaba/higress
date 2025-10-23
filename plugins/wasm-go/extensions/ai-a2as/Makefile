.PHONY: build test clean

PLUGIN_NAME = ai-a2as

build:
	tinygo build -o $(PLUGIN_NAME).wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer' main.go config.go rfc9421.go

test:
	go test -v ./...

clean:
	rm -f $(PLUGIN_NAME).wasm

.DEFAULT_GOAL := build


