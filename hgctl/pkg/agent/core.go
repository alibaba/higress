package agent

import (
	"os"
	"os/exec"
)

// integration with kode
type AICore struct {
	path string
}

func NewAICore(execPath string) *AICore {
	return &AICore{
		path: execPath,
	}
}

func (c *AICore) run(args ...string) error {
	cmd := exec.Command(AgentBinaryName, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()

}

func (c *AICore) Start() error {
	return c.run(AgentBinaryName)
}

// MCP Related
func (c *AICore) AddMCPServer(name string, url string) error {
	return c.run("mcp", "add-sse", name, url)
}
