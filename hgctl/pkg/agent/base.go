package agent

import (
	"fmt"
	"os"
)

const (
	AgentBinaryName  = "claude"
	BinaryVersion    = "0.1.0"
	DevVersion       = "dev"
	NodeLeastVersion = 18
	AgentInstallCmd  = "npm install -g @anthropic-ai/claude-code"
	AgentReleasePage = "https://docs.claude.com/en/docs/claude-code/setup"
)

// set up the core env
// 1. check if npm is installed
// 2. check the npm version
// 3. install hgctl-agent
func getAgent() *AgenticCore {
	if !checkAgentInstallStatus() {
		fmt.Println("⚠️ Prerequisites not satisfied. Exiting...")
		// exit directly
		os.Exit(1)
	}

	return NewAgenticCore()
}

func checkAgentInstallStatus() bool {
	// TODO: Support cross-platform:windows

	if !checkNodeInstall() {
		if err := promptNodeInstall(); err != nil {
			return false
		}
	}

	if !checkAgentInstall() {
		if err := promptAgentInstall(); err != nil {
			return false
		}
	}

	return true
}
