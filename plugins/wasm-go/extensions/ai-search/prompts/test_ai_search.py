import argparse
import requests
import time
import json

def main():
    # 解析命令行参数
    parser = argparse.ArgumentParser(description='AI Search Test Script')
    parser.add_argument('--question', required=True, help='The question to analyze')
    parser.add_argument('--prompt', required=True, help='The prompt file to analyze')
    parser.add_argument('--count', required=True, help='The max search count')
    args = parser.parse_args()

    # 读取并解析prompts.md模板
    # 这里假设prompts.md已经复制到当前目录
    with open(args.prompt, 'r', encoding='utf-8') as f:
        prompt_template = f.read()
    
    # 替换模板中的{question}变量
    prompt = prompt_template.replace('{question}', args.question)
    prompt = prompt_template.replace('{max_count}', args.count)

    # 准备请求数据
    headers = {
        'Content-Type': 'application/json',
    }
    data = {
        "model": "deepseek-v3",
        "max_tokens": 4096,
        "messages": [
            {
                "role": "user",
                "content": prompt
            }
        ]
    }

    # 发送请求并计时
    start_time = time.time()
    try:
        response = requests.post(
            'http://localhost:8080/v1/chat/completions', 
            headers=headers,
            data=json.dumps(data)
        )
        response.raise_for_status()
        end_time = time.time()

        # 处理响应
        result = response.json()
        print("Response:")
        print(result['choices'][0]['message']['content'])
        print(f"\nRequest took {end_time - start_time:.2f} seconds")
    except requests.exceptions.RequestException as e:
        print(f"Request failed: {e}")

if __name__ == '__main__':
    main()
