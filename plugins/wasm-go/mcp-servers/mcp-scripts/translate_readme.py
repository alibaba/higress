#!/usr/bin/env python3
import os
import sys
import json
import requests

def read_file(file_path):
    """Read file and return its content as a string."""
    try:
        with open(file_path, 'r', encoding='utf-8') as file:
            return file.read()
    except Exception as e:
        print(f"Error reading file: {e}")
        sys.exit(1)

def call_openai_api(content, base_url):
    """Call OpenAI API to translate content from Chinese to English."""
    url = f"http://{base_url}/chat/completions"
    
    # Prepare the prompt for OpenAI
    prompt = f"""
请将以下中文文档翻译成英文。保持原始的Markdown格式，包括标题、列表、代码块等。
确保翻译准确、专业，并且保持技术术语的正确性。

以下是需要翻译的中文文档：

{content}
"""
    
    # Prepare the API request
    headers = {
        "Content-Type": "application/json"
    }
    
    data = {
        "model": "gpt-4o",
        "messages": [
            {"role": "system", "content": "你是一个专业的技术文档翻译助手，擅长将中文技术文档翻译成英文。"},
            {"role": "user", "content": prompt}
        ],
        "temperature": 0.3
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
        print("Usage: python translate_readme.py <input_file_path> [output_file_path]")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2] if len(sys.argv) > 2 else "README.md"
    base_url = "127.0.0.1:8080/v1"
    
    # Read the Chinese content
    chinese_content = read_file(input_file)
    
    # Translate to English
    english_content = call_openai_api(chinese_content, base_url)
    
    # Save the translated content
    save_markdown(english_content, output_file)
    
    # Print the translated content to stdout
    print(english_content)

if __name__ == "__main__":
    main()
