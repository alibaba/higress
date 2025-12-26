# Agent Development Guide

Welcome to this AgentScope agent directory! This guide helps AI CLI tools (like Claude Code) understand the structure and assist you in building powerful agents.

## Directory Overview

This is an automatically generated agent directory with the following structure:

- **agent.py** - Main agent class (generated from agent.tmpl)
- **toolkit.py** - Agent's tools and MCP integrations (generated from toolkit.tmpl)
- **prompt.md** - User-provided system prompt for the agent
- **as_runtime_main.py** / **agentrun_main.py** - Deployment runtime files
- **agent.tmpl** / **toolkit.tmpl** / **agentscope.tmpl** - Generation templates

## What You Should Do

### Primary Focus: Improve Agent Intelligence

Your role is to help users build more capable, "agentic" agents by:

1. **Editing agent.py** - Enhance the agent class with:
   - Custom reasoning logic
   - Agent-specific hooks and behaviors
   - Memory management strategies
   - Multi-step task handling

2. **Editing toolkit.py** - Expand agent capabilities by:
   - Adding new tool functions
   - Integrating MCP (Model Context Protocol) servers
   - Configuring tool access and permissions

3. **Editing prompt.md** (when requested) - Refine the system prompt to:
   - Improve agent behavior and personality
   - Add domain-specific instructions
   - Define task-specific guidelines

### Critical Constraints

**DO NOT MODIFY** these deployment files:
- `as_runtime_main.py`
- `agentrun_main.py`

These files handle agent deployment and runtime orchestration. They are managed by the agent framework and should not be changed during development.

## Learning AgentScope

Before helping users, you should become proficient with AgentScope:

### Use the DeepWiki MCP Server

You have access to the `mcp-deepwiki` server. Use it to learn about AgentScope:

```python
# Query the AgentScope repository
ask_question(
    repoName="agentscope-ai/agentscope",
    question="How does the ReActAgent work?"
)
```

Study these key concepts:
- ReActAgent architecture (Reasoning + Acting loop)
- Agent hooks and lifecycle methods
- Toolkit and tool registration
- Memory systems (short-term and long-term)
- Message formatting and model integration
- MCP integration for external tools

### Testing Your Agent

Use the `agentscope-test-runner` subagent to test agent functionality:

```python
# Launch test runner to validate agent behavior
Task(
    subagent_type="agentscope-test-runner",
    prompt="Test the agent's ability to handle multi-step tasks",
    description="Testing agent functionality"
)
```

**Don't** write your own test harness - use this specialized subagent.

## Building Great Agents: Examples

### Example 1: Browser Automation Agent

Based on the AgentScope BrowserAgent, here's how to build a specialized web agent:

**Key Patterns:**

1. **Extend ReActAgent** - Inherit from ReActAgent for reasoning-acting loop
2. **Use Hooks** - Register instance hooks to customize behavior at different lifecycle points:
   - `pre_reply` - Run before generating responses
   - `pre_reasoning` - Execute before reasoning phase
   - `post_reasoning` - Execute after reasoning phase
   - `post_acting` - Execute after taking actions

3. **Manage Memory** - Implement memory summarization to prevent context overflow
4. **Leverage MCP Tools** - Connect to MCP servers (like Playwright browser tools) via toolkit

```python
class Agent(ReActAgent):
    def __init__(self, name, model, formatter, memory, toolkit, ...):
        super().__init__(name, sys_prompt, model, formatter, memory, toolkit, max_iters)

        # Register custom hooks
        self.register_instance_hook(
            "pre_reply",
            "custom_hook_name",
            custom_hook_function
        )
```

### Example 2: Research Agent

For research and analysis tasks:

**Key Features:**
- Knowledge base integration for RAG (Retrieval-Augmented Generation)
- Long-term memory for persistent context
- Plan notebook for complex multi-step research
- Query rewriting for better information retrieval

```python
class Agent(ReActAgent):
    def __init__(
        self,
        name,
        sys_prompt,
        model,
        formatter,
        toolkit,
        memory,
        long_term_memory=None,
        knowledge=None,
        enable_rewrite_query=True,
        plan_notebook=None,
        ...
    ):
        # Initialize with research-focused capabilities
        super().__init__(...)
```

### Example 3: Code Assistant Agent

For software development tasks:

**Key Capabilities:**
- File operation tools (read, write, insert)
- Code execution (execute_python_code, execute_shell_command)
- Image/audio processing for multimodal interactions
- MCP integration for IDE tools

### Common Agent Patterns

1. **Tool Registration** (in toolkit.py):
```python
from agentscope.tool import Toolkit
from agentscope.tool import execute_shell_command, view_text_file

toolkit = Toolkit()
toolkit.register_tool_function(execute_shell_command)
toolkit.register_tool_function(view_text_file)
```

2. **MCP Integration** (in toolkit.py):
```python
from agentscope.mcp import HttpStatelessClient

async def register_mcp(toolkit):
    client = HttpStatelessClient(
        name="browser-tools",
        transport="sse",
        url="http://localhost:3000/sse"
    )
    await toolkit.register_mcp_client(client)
```

3. **Custom Hooks** (in agent.py):
```python
async def pre_reasoning_hook(self, *args, **kwargs):
    """Custom logic before reasoning"""
    # Add context, check conditions, etc.
    pass

# In __init__:
self.register_instance_hook("pre_reasoning", "my_hook", pre_reasoning_hook)
```

## More Examples and Resources

Explore official AgentScope examples:
- https://github.com/modelscope/agentscope/tree/main/examples/agent

Key examples to study:
- **ReAct Agent** - Basic reasoning-acting agent
- **Conversation Agent** - Multi-turn dialogue handling
- **User Agent** - Human-in-the-loop interactions
- **Tool Agent** - Advanced tool usage patterns

## Development Workflow

1. **Understand Requirements** - Clarify what the agent should do
2. **Learn Patterns** - Use DeepWiki to research relevant AgentScope patterns
3. **Design Agent** - Choose base class and required capabilities
4. **Implement in agent.py** - Write custom agent logic
5. **Add Tools in toolkit.py** - Register needed tools and MCP servers
6. **Test with agentscope-test-runner** - Validate functionality
7. **Iterate** - Refine based on test results

## Best Practices

1. **Start Simple** - Begin with basic ReActAgent, add complexity as needed
2. **Use Hooks Wisely** - Don't overcomplicate; hooks should have clear purposes
3. **Memory Management** - Implement summarization for long conversations
4. **Tool Selection** - Only add tools the agent actually needs
5. **Clear Prompts** - Write specific, actionable system prompts in prompt.md
6. **Test Iteratively** - Use the test-runner frequently during development

## Getting Help

- Use DeepWiki MCP to query AgentScope documentation
- Study the browser_agent.py example in this guide
- Reference official examples at https://github.com/agentscope-ai/agentscope
- Test early and often with agentscope-test-runner

---

**Remember:** Focus on making the agent intelligent and capable. The deployment infrastructure is already handled - your job is to build the "brain" of the agent in agent.py and give it the right "tools" in toolkit.py.
