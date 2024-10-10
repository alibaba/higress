BUILD_OPTS="--release"

.DEFAULT:
build:
	rustup target add wasm32-wasi
	cargo build --target wasm32-wasi ${BUILD_OPTS}
	find target -name "*.wasm" -d 3 -exec cp "{}" plugin.wasm \;

clean:
	cargo clean
	rm -f plugin.wasm
