## 介绍

此 SDK 用于使用 AssemblyScript 语言开发 Higress 的 Wasm 插件。

### 如何使用SDK

创建一个新的 AssemblyScript 项目

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

将`"@higress/proxy-wasm-assemblyscript-sdk": "^0.0.1"`和`"@higress/wasm-assemblyscript": "^0.0.1"`添加到你的依赖项中，然后运行`npm install`。

### 本地构建

```
npm run asbuild
```

构建结果将在`build`文件夹中。其中，`debug.wasm`和`release.wasm`是已编译的文件。