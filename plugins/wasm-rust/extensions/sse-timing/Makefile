BUILD_OPTS="--release"

.DEFAULT:
build:
	cargo build --target wasm32-wasi ${BUILD_OPTS}

copy: build
	find target -name "*.wasm" -d 3 -exec cp "{}" plugin.wasm \;

clean:
	@cargo clean

docker-compose: copy
	@docker-compose up
