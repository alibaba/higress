package agent

import (
	"os"
	"os/exec"
)

type AgenticCore struct {
}

func NewAgenticCore() *AgenticCore {
	return &AgenticCore{}
}

func (c *AgenticCore) run(args ...string) error {
	cmd := exec.Command(AgentBinaryName, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()

}

// ------- Initialization  -------
func (c *AgenticCore) Start() error {
	return c.run(AgentBinaryName)
}

// ------- MCP  -------
func (c *AgenticCore) AddMCPServer(name string, url string) error {
	return c.run("mcp", "add", "--transport", HTTP, name, url)
}
