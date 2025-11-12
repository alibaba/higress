#!/usr/bin/env python3
"""
Terminal-Bench Agent for Higress AI-Proxy Integration Testing
This agent communicates with LLM through Higress AI-Proxy gateway
"""

import os
import json
import requests
import argparse
from typing import List, Dict, Any, Optional


class HigressAgent:
    """Agent that communicates with LLM through Higress AI-Proxy"""
    
    def __init__(self, 
                 higress_endpoint: str,
                 api_key: str,
                 model: str = "deepseek-chat",
                 temperature: float = 0.7,
                 max_tokens: int = 4096):
        """
        Initialize Higress Agent
        
        Args:
            higress_endpoint: Higress AI-Proxy endpoint (e.g., http://localhost:8080/v1/chat/completions)
            api_key: API key for authentication
            model: Model name
            temperature: Temperature for response generation
            max_tokens: Maximum tokens for response
        """
        self.endpoint = higress_endpoint
        self.api_key = api_key
        self.model = model
        self.temperature = temperature
        self.max_tokens = max_tokens
        self.conversation_history: List[Dict[str, Any]] = []
        
    def add_message(self, role: str, content: str):
        """Add a message to conversation history"""
        self.conversation_history.append({
            "role": role,
            "content": content
        })
        
    def add_tool_result(self, tool_call_id: str, result: str):
        """Add tool execution result"""
        self.conversation_history.append({
            "role": "tool",
            "tool_call_id": tool_call_id,
            "content": result
        })
        
    def call_llm(self, 
                 messages: Optional[List[Dict[str, Any]]] = None,
                 tools: Optional[List[Dict[str, Any]]] = None) -> Dict[str, Any]:
        """
        Call LLM through Higress AI-Proxy
        
        Args:
            messages: Messages to send (if None, use conversation history)
            tools: Tool definitions
            
        Returns:
            Response from LLM
        """
        if messages is None:
            messages = self.conversation_history
            
        payload = {
            "model": self.model,
            "messages": messages,
            "temperature": self.temperature,
            "max_tokens": self.max_tokens
        }
        
        if tools:
            payload["tools"] = tools
            
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.api_key}"
        }
        
        try:
            response = requests.post(
                self.endpoint,
                json=payload,
                headers=headers,
                timeout=60
            )
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"Error calling Higress AI-Proxy: {e}")
            if hasattr(e, 'response') and e.response is not None:
                print(f"Response: {e.response.text}")
            raise
            
    def run_task(self, task_prompt: str, tools: Optional[List[Dict[str, Any]]] = None) -> str:
        """
        Run a task with the agent
        
        Args:
            task_prompt: Task description
            tools: Available tools
            
        Returns:
            Final response from agent
        """
        # Add task prompt to conversation
        self.add_message("user", task_prompt)
        
        max_iterations = 20  # Prevent infinite loops
        iteration = 0
        
        while iteration < max_iterations:
            iteration += 1
            print(f"\n=== Iteration {iteration} ===")
            
            # Call LLM
            response = self.call_llm(tools=tools)
            
            # Extract response
            if "choices" not in response or len(response["choices"]) == 0:
                print("No response from LLM")
                break
                
            choice = response["choices"][0]
            message = choice.get("message", {})
            
            # Add assistant message to history
            self.conversation_history.append(message)
            
            # Check if there are tool calls
            tool_calls = message.get("tool_calls", [])
            if not tool_calls:
                # No tool calls, task completed
                content = message.get("content", "")
                print(f"Agent response: {content}")
                return content
                
            # Execute tool calls
            print(f"Agent requested {len(tool_calls)} tool call(s)")
            for tool_call in tool_calls:
                tool_name = tool_call["function"]["name"]
                tool_args = json.loads(tool_call["function"]["arguments"])
                tool_id = tool_call["id"]
                
                print(f"Tool call: {tool_name}({tool_args})")
                
                # Execute tool (this should be implemented by the environment)
                # For terminal-bench, this would be handled by the framework
                # Here we just add placeholder
                result = f"Tool {tool_name} executed with args {tool_args}"
                self.add_tool_result(tool_id, result)
                
        print("Max iterations reached")
        return "Task did not complete within max iterations"


def main():
    parser = argparse.ArgumentParser(description="Higress Terminal-Bench Agent")
    parser.add_argument("--endpoint", required=True, help="Higress AI-Proxy endpoint")
    parser.add_argument("--api-key", required=True, help="API key")
    parser.add_argument("--model", default="deepseek-chat", help="Model name")
    parser.add_argument("--task", help="Task prompt (for testing)")
    
    args = parser.parse_args()
    
    agent = HigressAgent(
        higress_endpoint=args.endpoint,
        api_key=args.api_key,
        model=args.model
    )
    
    if args.task:
        # Test mode
        result = agent.run_task(args.task)
        print(f"\nFinal result: {result}")
    else:
        print("Agent initialized. Use with terminal-bench framework.")


if __name__ == "__main__":
    main()

