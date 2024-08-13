## 介绍

此 SDK 用于使用 AssemblyScript 语言开发 Higress 的 Wasm 插件。

### 如何使用SDK

创建一个新的 AssemblyScript 项目。

```
npm init
npm install --save-dev assemblyscript
npx asinit .
```

在asconfig.json文件中，作为传递给asc编译器的选项之一，包含"use": "abort=abort_proc_exit"。

```
{
  "options": {
    "use": "abort=abort_proc_exit"
  }
}
```

将`"@higress/proxy-wasm-assemblyscript-sdk": "^0.0.2"`和`"@higress/wasm-assemblyscript": "^0.0.4"`添加到你的依赖项中，然后运行`npm install`。

### 本地构建

```
npm run asbuild
```

构建结果将在`build`文件夹中。其中，`debug.wasm`和`release.wasm`是已编译的文件，在生产环境中建议使用`release.wasm`。

注：如果需要插件带有 name section 信息需要带上`"debug": true`，编译参数解释详见[using-the-compiler](https://www.assemblyscript.org/compiler.html#using-the-compiler)。

```json
"release": {
  "outFile": "build/release.wasm",
  "textFile": "build/release.wat",
  "sourceMap": true,
  "optimizeLevel": 3,
  "shrinkLevel": 0,
  "converge": false,
  "noAssert": false,
  "debug": true
}
```

### AssemblyScript 限制

此 SDK 使用的 AssemblyScript 版本为`0.27.29`，参考[AssemblyScript Status](https://www.assemblyscript.org/status.html)该版本尚未支持闭包、异常、迭代器等特性，并且JSON，正则表达式等功能还尚未在标准库中实现，暂时需要使用社区提供的实现。

