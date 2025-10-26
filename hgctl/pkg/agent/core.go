// Copyright (c) 2025 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
