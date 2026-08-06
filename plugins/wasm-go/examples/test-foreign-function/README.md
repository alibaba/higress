## 功能说明

此插件示例用于展示在无响应body情况下如何添加body数据。

**注意**
1. 原始响应不能够有body，如果有原始的响应body，会造成网关crash
2. header阶段需返回`types.ActionPause`
3. `Endstream`必须设置为`true`

示例中 `inject_encoded_data_to_filter_chain_on_header` 该函数是异步调用，需要保证调用时流不被销毁，有body或者header阶段不返回`types.ActionPause`都可能导致流被提前销毁。

一份无响应body的flask代码示例：

```python
import os
from flask import Flask, request, Response

app = Flask(__name__)

@app.route('/test', methods=['GET', 'POST'])
def print_request():
    return Response(status=200)

if __name__ == '__main__':
    app.run("0.0.0.0", 5000, debug=False)
```