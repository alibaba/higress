// Simple MCP Server for Nginx Migration Tools - Standalone Mode
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"nginx-migration-mcp-final/internal/standalone"
)

func main() {
	config := standalone.LoadConfig()
	server := standalone.NewMCPServer(config)

	// 只在调试模式下输出启动日志
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
		log.Println("🚀 Nginx迁移MCP服务器启动...")
		log.Println("🔗 等待MCP客户端连接...")
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg standalone.MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
				log.Printf("JSON解析错误: %v", err)
			}
			continue
		}

		response := server.HandleMessage(msg)

		responseBytes, _ := json.Marshal(response)
		fmt.Println(string(responseBytes))
	}
}
