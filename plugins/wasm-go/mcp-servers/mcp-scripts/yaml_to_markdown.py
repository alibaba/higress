#!/usr/bin/env python3
import os
import sys
import json
import requests

def read_yaml_file(file_path):
    """Read YAML file and return its content as a string."""
    try:
        with open(file_path, 'r', encoding='utf-8') as file:
            return file.read()
    except Exception as e:
        print(f"Error reading YAML file: {e}")
        sys.exit(1)

def call_openai_api(yaml_content, base_url):
    """Call OpenAI API to transform YAML to Markdown."""
    url = f"http://{base_url}/chat/completions"
    
    # Prepare the prompt for OpenAI
    prompt = f"""
请将以下MCP服务器YAML配置转换为Markdown格式的功能简介文档。
文档应包含两个二级标题：
1. ## 功能简介 - 概述该MCP服务器的主要功能和用途
2. ## 工具简介 - 概括介绍每个工具的用途和使用场景

以下是YAML配置内容：

{yaml_content}
"""
    
    # Prepare the API request
    headers = {
        "Content-Type": "application/json"
    }
    
    data = {
        "model": "gpt-4",
        "messages": [
            {"role": "system", "content": "你是一个专业的技术文档编写助手，擅长将技术配置文件转换为易于理解的文档。"},
            {"role": "user", "content": prompt}
        ],
        "temperature": 0.7
    }
    
    try:
        response = requests.post(url, headers=headers, json=data)
        response.raise_for_status()
        
        result = response.json()
        if "choices" in result and len(result["choices"]) > 0:
            return result["choices"][0]["message"]["content"]
        else:
            print("Error: Unexpected API response format")
            sys.exit(1)
    except Exception as e:
        print(f"Error calling OpenAI API: {e}")
        sys.exit(1)

def save_markdown(markdown_content, output_file):
    """Save the Markdown content to a file."""
    try:
        with open(output_file, 'w', encoding='utf-8') as file:
            file.write(markdown_content)
    except Exception as e:
        print(f"Error saving Markdown file: {e}")
        sys.exit(1)

def main():
    if len(sys.argv) < 2:
        print("Usage: python yaml_to_markdown.py <yaml_file_path> [output_file_path]")
        sys.exit(1)
    
    yaml_file = sys.argv[1]
    output_file = sys.argv[2] if len(sys.argv) > 2 else "mcp-server-docs.md"
    base_url = "127.0.0.1:8080/v1"
    
    yaml_content = read_yaml_file(yaml_file)
    markdown_content = call_openai_api(yaml_content, base_url)
    save_markdown(markdown_content, output_file)
    
    # Print the Markdown content to stdout
    print(markdown_content)

if __name__ == "__main__":
    main()
