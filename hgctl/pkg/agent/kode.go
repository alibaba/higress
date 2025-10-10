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
	// check if there is kode path
	// TODO: environment configuration(put kode or cli.js in PATH)
	return &KodeClient{
		path: execPath,
	}
}

func (c *KodeClient) Run(args []string) error {
	cmd := exec.Command(c.path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()

}

func (c *KodeClient) AddMCPServer(name string, url string) error {
	return c.Run([]string{
		"mcp", "add", name, url,
	})
}
