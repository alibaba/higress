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
Respond to the human as helpfully and accurately as possible. You have access to the following tools:

{{tools_desc}}

Use a json blob to specify a tool by providing an action key (tool name) and an action_input key (tool input).
Valid "action" values: "Final Answer" or {{tool_names}}

Provide only ONE action per $JSON_BLOB, as shown:

```

	{
	  "action": $TOOL_NAME,
	  "action_input": $ACTION_INPUT
	}

```

Follow this format:

Question: input question to answer
Thought: consider previous and subsequent steps
Action:
```
$JSON_BLOB
```
Observation: action result
... (repeat Thought/Action/Observation N times)
Thought: I know what to respond
Action:
```

	{
	  "action": "Final Answer",
	  "action_input": "Final response to human"
	}

```

Begin! Reminder to ALWAYS respond with a valid json blob of a single action. Use tools if necessary. Respond directly if appropriate. Format is Action:```$JSON_BLOB```then Observation:.
{{historic_messages}}
Question: {{query}}
*/
const EN_Template = `
Respond to the human as helpfully and accurately as possible.You have access to the following tools:

%s

Use a json blob to specify a tool by providing an action key (tool name) and an action_input key (tool input).
Valid "action" values: "Final Answer" or %s

Provide only ONE action per $JSON_BLOB, as shown:
` + "```" + `
{
  "action": $TOOL_NAME,
  "action_input": $ACTION_INPUT
}
` + "```" + `
Follow this format:
Question: %s
Thought: %s 
Action: ` + "```" + `$JSON_BLOB` + "```" + `

Observation: %s 
... (repeat Thought/Action/Observation N times)
Thought: %s
Action:` + "```" + `
{
  "action": "Final Answer",
  "action_input": "Final response to human"
}
` + "```" + `
Begin! Reminder to ALWAYS respond with a valid json blob of a single action. Use tools if necessary. Respond directly if appropriate.Format is Action:` + "```" + `$JSON_BLOB` + "```" + `then Observation:.
%s
Question: %s
`

/*
尽可能帮助和准确地回答人的问题。您可以使用以下工具：

{tool_descs}

使用 json blob，通过提供 action key（工具名称）和 action_input key（工具输入）来指定工具。
有效的 "action"值为 "Final Answer"或 {tool_names}

每个 $JSON_BLOB 只能提供一个操作，如图所示：

```

	{{
	  "action": $TOOL_NAME,
	  "action_input": $ACTION_INPUT
	}}

```

按照以下格式:
Question: 输入要回答的问题
Thought: 考虑之前和之后的步骤
Action:
```
$JSON_BLOB
```

Observation: 行动结果
...（这个Thought/Action//Observation可以重复N次）
Thought: 我知道该回应什么
Action:
```

	{{
	  "action": "Final Answer",
	  "action_input": "Final response to human"
	}}

```

开始！提醒您始终使用单个操作的有效 json blob 进行响应。必要时使用工具。如果合适，可直接响应。格式为 Action:```$JSON_BLOB```then Observation:.
{historic_messages}
Question: {input}
*/
const CH_Template = `
尽可能帮助和准确地回答人的问题。您可以使用以下工具：

%s

使用 json blob，通过提供 action key（工具名称）和 action_input key（工具输入）来指定工具。
有效的 "action"值为 "Final Answer"或 %s

每个 $JSON_BLOB 只能提供一个操作，如图所示：
` + "```" + `
{
  "action": $TOOL_NAME,
  "action_input": $ACTION_INPUT
}
` + "```" + `
按照以下格式:
Question: %s
Thought: %s 
Action: ` + "```" + `$JSON_BLOB` + "```" + `

Observation: %s 
...（这个Thought/Action//Observation可以重复N次）
Thought: %s
Action:` + "```" + `
{
  "action": "Final Answer",
  "action_input": "Final response to human"
}
` + "```" + `
开始！提醒您始终使用单个操作的有效 json blob 进行响应。必要时使用工具。如果合适，可直接响应。格式为 Action:` + "```" + `$JSON_BLOB` + "```" + `then Observation:.
%s
Question: %s
`
