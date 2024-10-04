## 介绍

此 SDK 用于使用 Rust 语言开发 Higress 的 Wasm 插件。

### 创建新插件

在当前文件夹下（`plugins/wasm-rust`）执行下面的命令，将`${...}`替换为新插件的内容，其中`--name`为必填项。

```shell
sh ./scripts/gen.sh \
  --name ${NAME} \
  --keywords ${KEYWORDS} \
  --description ${DESCRIPTION} \
  --testing \
  --testing-port ${PORT} 
```

执行完成后将会在`plugins/wasm-rust/extensions/${NAME}`目录下生成以下文件，其中`src/lib.rs`中的内容即为插件需要实现的逻辑

```tree
.
├── Cargo.toml
├── Makefile
├── README.md
├── docker-compose.yaml
├── envoy.yaml
├── plugin.wasm
└── src
    └── lib.rs
```

执行`make docker-compose`可以在docker中运行higress实例，并通过`curl localhost:${PORT}`来测试插件是否正常运行