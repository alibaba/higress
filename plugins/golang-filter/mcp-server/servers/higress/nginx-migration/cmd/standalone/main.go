// Nginx Migration MCP Server - Standalone Mode
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"nginx-migration-mcp/internal/standalone"
)

func main() {
	config := standalone.LoadConfig()
	server := standalone.NewMCPServer(config)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg standalone.MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		response := server.HandleMessage(msg)

		responseBytes, _ := json.Marshal(response)
		fmt.Println(string(responseBytes))
	}
}
