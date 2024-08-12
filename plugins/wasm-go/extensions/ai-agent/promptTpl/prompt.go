package prompttpl

// input param
// {name_for_model}
// {description_for_model}
// {description_for_model}
// {description_for_model}
// {parameters}
const TOOL_DESC = `
%s: Call this tool to interact with the %s API. What is the %s API useful for? %s 
Parameters: 
%s 
Format the arguments as a JSON object.`

/*
Answer the following questions as best you can, but speaking as a pirate might speak. You have access to the following tools:

%s

Use the following format:

Question: the input question you must answer
Thought: you should always think about what to do
Action: the action to take, should be one of %s
Action Input: the input to the action
Observation: the result of the action
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: I now know the final answer
Final Answer: the final answer to the original input question, please give the most direct answer directly in Chinese, not English, and do not add extra content.

Begin! Remember to speak as a pirate when giving your final answer. Use lots of "Arg"s

Question: %s
*/
const EN_Template = `
Answer the following questions as best you can, but speaking as a pirate might speak. You have access to the following tools:

%s

Use the following format:

Question: %s
Thought: %s
Action: the action to take, should be one of %s
Action Input: %s
Observation: %s
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: %s
Final Answer: %s

%s

Question: %s
`

/*
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
*/
const CH_Template = `
尽你所能回答以下问题。你可以使用以下工具：

%s

请使用以下格式，其中Action字段后必须跟着Action Input字段，并且不要将Action Input替换成Input或者tool等字段，不能出现格式以外的字段名，每个字段在每个轮次只出现一次：
Question: %s
Thought: %s 
Action: 要采取的动作，动作只能是%s中的一个 ，一定不要加入其它内容
Action Input: %s
Observation: %s 
...（这个Thought/Action/Action Input/Observation可以重复N次） 
Thought: %s
Final Answer: %s

%s

Question: %s
`
