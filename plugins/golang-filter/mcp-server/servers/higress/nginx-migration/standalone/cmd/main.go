package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"nginx-migration-mcp/standalone"
)

const Version = "1.0.0"

func main() {
	// Load config
	config := standalone.LoadConfig()

	server := standalone.NewMCPServer(config)

	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()

		var msg standalone.MCPMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		response := server.HandleMessage(msg)
		responseBytes, _ := json.Marshal(response)

		writer.Write(responseBytes)
		writer.WriteByte('\n')
		writer.Flush()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
		os.Exit(1)
	}
}
