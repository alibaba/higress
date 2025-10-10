package agent

import (
	"os"
	"os/exec"
)

// integration with kode
type KodeClient struct {
	path string
}

func NewKodeClient(execPath string) *KodeClient {
	return &KodeClient{
		path: execPath,
	}
}

func (c *KodeClient) run(args []string) error {
	cmd := exec.Command(AgentBinaryName, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()

}

func (c *KodeClient) AddMCPServer(name string, url string) error {
	return c.run([]string{
		"mcp", "add-sse", name, url,
	})
}
