## 功能说明

此插件示例用于展示在无响应body情况下如何添加body数据。

**注意**
1. 原始响应不能够有body，如果有原始的响应body，会造成网关crash
2. header阶段需返回`types.HeaderContinueAndEndStream`
3. `Endstream`必须设置为`true`


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