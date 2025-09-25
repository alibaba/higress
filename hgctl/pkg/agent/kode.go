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

func (c *KodeClient) Run(args []string) error {
	cmd := exec.Command(c.path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdin = os.Stdin
	cmd.Stdin = os.Stdin

	return cmd.Run()

}

func (c *KodeClient) AddMCPServer(name string, url string) error {
	return c.Run([]string{
		"mcp", "add", name, url,
	})
}
