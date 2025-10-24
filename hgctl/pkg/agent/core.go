package agent

import (
	"os"
	"os/exec"
)

type AgenticCore struct {
	path string
}

func NewAgenticCore(execPath string) *AgenticCore {
	return &AgenticCore{
		path: execPath,
	}
}

func (c *AgenticCore) run(args ...string) error {
	cmd := exec.Command(AgentBinaryName, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()

}

func (c *AgenticCore) Start() error {
	return c.run(AgentBinaryName)
}

// MCP Related
func (c *AgenticCore) AddMCPServer(name string, url string) error {
	return c.run("mcp", "add-sse", name, url)
}
