package prompttpl

// input param
// {name_for_model}
// {name_for_human}
// {name_for_human}
// {description_for_model}
// {parameters}
const TOOL_DESC = `%s: Call this tool to interact with the %s API. What is the %s API useful for? %s Parameters: %s Format the arguments as a JSON object.`

// input param
// TOOL_DESC
// {Tool_name}
// {input}
const Template = `
尽你所能回答以下问题。你可以使用以下工具：

%s

请使用以下格式，其中Action字段后必须跟着Action Input字段，并且不要将Action Input替换成Input或者tool等字段，不能出现格式以外的字段名，每个字段在每个轮次只出现一次：

Question: 你需要回答的输入问题 

Thought: 你应该总是思考该做什么 

Action: 要采取的动作，动作只能是%s中的一个 ，一定不要加入其它内容

Action Input: 行动的输入，必须出现在Action后。

Observation: 行动的结果 

...（这个Thought/Action/Action Input/Observation可以重复N次） 

Thought: 我现在知道最终答案 

Final Answer: 对原始输入问题的最终答案

再次重申，不要修改以上模板的字段名称，开始吧！

Question: %s
`
