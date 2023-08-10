# 功能说明
`chatgpt-proxy`插件实现了代理请求AI大语言模型服务的功能。

# 配置字段

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
|  model     |  string     | 选填   |   text-davinci-003  |  配置使用的模型模型名称   |
|  apiKey   |  string     | 必填   |   -  |  配置使用的OpenAI API密钥   |
|  promptParam     |  string     | 选填  |   prompt  |  配置prompt的来源字段名称，URL参数   |
|  chatgptUri     |  string     | 选填     |  api.openai.com/v1/completions   |  配置调用AI模型服务的URL路径，默认值为OPENAI的API调用路径   |
# 配置示例

## 进行OpenAI curie模型的调用
```yaml
apiKey: "xxxxxxxxxxxxxx",
promptParam: "text",
model: "curie"
```


