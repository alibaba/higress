package agent

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	AgentBinaryName = "hgctl-agent"
	BinaryVersion   = "0.1.0"
	DevVersion      = "dev"
)

// set up the kode env
// 1. npm install
// 2. check the npm version
// 3. install hgctl-agent
func getClient() *KodeClient {
	if !checkAgentInstallStatus() {
		fmt.Println("installing......")
		if err := installAgent(); err != nil {
			fmt.Printf("failed to install agent: %s\n", err)
			panic("failed to launch hgctl-agent, install failed")
		}
	}

	return NewKodeClient("")

}

func installAgent() error {
	args := []string{"install", "--verbose", "-g", fmt.Sprintf("%s@%s", AgentBinaryName, DevVersion)}
	cmd := exec.Command("npm", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}
	return nil
}

func checkAgentInstallStatus() bool {
	// TODO: Support cross-platform:windows
	cmd := exec.Command(AgentBinaryName, "--version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("failed to launch hgctl-agent output: %s due to: %v", string(output), err)
		return false
	}
	return true
}
