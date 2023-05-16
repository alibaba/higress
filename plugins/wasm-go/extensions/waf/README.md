# Building the filter

```bash
go run mage.go build
```

You will find the WASM plugin under `./build/main.wasm`.


# Local test
```bash
cd build
docker compose up
```