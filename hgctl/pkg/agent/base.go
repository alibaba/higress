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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/alibaba/higress/hgctl/pkg/agent/common"
	"github.com/alibaba/higress/hgctl/pkg/agent/services"
	"github.com/fatih/color"
	"github.com/spf13/viper"
)

const (
	NodeLeastVersion = 18
)

type HimarketAdminAuthArg struct {
	hmURL      string
	hmUser     string
	hmPassword string
}

// Developer's page
type HimarketDevAuthArg struct {
	hmURL      string
	hmUser     string
	hmPassword string
}

func (h *HimarketAdminAuthArg) validate() error {
	if h.hmURL == "" || h.hmUser == "" || h.hmPassword == "" {
		return fmt.Errorf("invalid args")
	}
	return nil
}

type HigressConsoleAuthArg struct {
	// higress console auth arg
	hgURL      string
	hgUser     string
	hgPassword string
}

func (h *HigressConsoleAuthArg) validate() error {
	if h.hgURL == "" || h.hgUser == "" || h.hgPassword == "" {
		fmt.Println("--higress-console-user, --higress-console-url, --higress-console-password must be provided")
		return fmt.Errorf("invalid args")
	}
	return nil
}

func init() {
	// Init the global configuration from config file
	InitConfig()
}

func resolveHimarketAdminAuth(arg *HimarketAdminAuthArg) {
	if arg.hmURL == "" {
		arg.hmURL = viper.GetString(HIMARKET_ADMIN_URL)
	}
	if arg.hmUser == "" {
		arg.hmUser = viper.GetString(HIMARKET_ADMIN_USER)
	}
	if arg.hmPassword == "" {
		arg.hmPassword = viper.GetString(HIMARKET_ADMIN_PASSWORD)
	}
}

// resolve from viper
func resolveHigressConsoleAuth(arg *HigressConsoleAuthArg) {
	if arg.hgURL == "" {
		arg.hgURL = viper.GetString(HIGRESS_CONSOLE_URL)
	}
	if arg.hgUser == "" {
		arg.hgUser = viper.GetString(HIGRESS_CONSOLE_USER)
	}
	if arg.hgPassword == "" {
		arg.hgPassword = viper.GetString(HIGRESS_CONSOLE_PASSWORD)
	}

	// fmt.Printf("arg: %v\n", arg)

	if arg.hgUser == "" || arg.hgPassword == "" {
		// Here we do not return this error, because it will failed when validate arg
		if err := tryToGetLocalCredential(arg); err != nil {
			fmt.Printf("failed to get local higress console credential: %s\n", err)
		}
	}
}

func parseTypeToAPIProductType(typ string) string {
	switch typ {
	case "a2a":
		return string(common.AGENT_API)
	case "restful":
		return string(common.REST_API)
	case "model":
		return string(common.MODEL_API)
	case "mcp":
		return string(common.MCP_SERVER)
	default:
		return ""
	}
}

// This function serves MCP API as well as Model API for now.
func publishAPIToHimarket(typ, name string, arg HimarketAdminAuthArg) error {

	if err := arg.validate(); err != nil {
		return err
	}

	client := services.NewHimarketClient(arg.hmURL, arg.hmUser, arg.hmPassword)

	productName := fmt.Sprintf("%s-%s", typ, name)

	var gatewayId = viper.GetString(HIMARKET_TARGET_HIGRESS_ID)
	prompt := survey.Input{
		Message: fmt.Sprintf("Enter the target Higress instance id on Himarket(%s):", gatewayId),
		Default: gatewayId,
		Help:    fmt.Sprintf("refers to %s/consoles/gateway to get your target Higress instance's id", arg.hmURL),
	}

	if err := survey.AskOne(&prompt, &gatewayId); err != nil {
		return fmt.Errorf("failed to get target higress gatewayID: %s", err)
	}

	body := services.BuildAPIProductBody(productName, "An agent API import by hgctl", parseTypeToAPIProductType(typ))
	resp, err := services.HandleAddAPIProduct(client, body)
	if err != nil {
		fmt.Println(resp)
		return err
	}

	product_id := string(resp)
	var refBody map[string]interface{}

	if typ == "mcp" {
		refBody = services.BuildRefMCPAPIProductBody(gatewayId, product_id, name)
	} else {
		// target_route is the route_name in Higress, refers to `publishAgentAPIToHigress`
		target_route := fmt.Sprintf("%s-route", name)
		refBody = services.BuildRefModelAPIProductBody(gatewayId, product_id, target_route)

	}

	if resp, err := services.HandleRefAPIProduct(client, product_id, refBody); err != nil {
		fmt.Println(string(resp))
		return err
	}

	return nil
}

// use pre-defined command /gen-agent to generate sys prompt
func generateAgentPromptByCore(desc string) (string, error) {
	core := NewAgenticCore()
	prompt, err := core.runWithResult(fmt.Sprintf("/gen-agent %s", desc), "--print")
	if err != nil {
		return "", err
	}
	return prompt, nil
}

type EnvProvisioner struct {
	core        CoreType
	installCmd  string
	releasePage string

	// ~/.<core>
	dirName string
}

func getCore() (*AgenticCore, error) {
	provisioner := EnvProvisioner{
		core: CoreType(viper.GetString(HGCTL_AGENT_CORE)),
	}

	if err := provisioner.check(); err != nil {
		return nil, fmt.Errorf("⚠️ Prerequisites not satisfied: %s Exiting...", err)
	}

	return NewAgenticCore(), nil
}

func (p *EnvProvisioner) init() {
	switch p.core {
	case CORE_QODERCLI:
		p.installCmd = "npm install -g @qoder-ai/qodercli"
		p.releasePage = "https://docs.qoder.com/zh/cli/quick-start"
		p.dirName = "qoder"

	case CORE_CLAUDE:
		p.installCmd = "npm install -g @anthropic-ai/claude-code"
		p.releasePage = "https://docs.claude.com/en/docs/claude-code/setup"
		p.dirName = "claude"
	}
}

func (p *EnvProvisioner) check() error {
	p.init()

	if !p.checkNodeInstall() {
		if err := p.promptNodeInstall(); err != nil {
			return err
		}
	}

	if !p.checkAgentInstall() {
		if err := p.promptAgentInstall(); err != nil {
			return err
		}
	}
	return nil
}

func (p *EnvProvisioner) checkNodeInstall() bool {
	cmd := exec.Command("node", "-v")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	versionStr := strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
	parts := strings.Split(versionStr, ".")
	if len(parts) == 0 {
		return false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}

	return major >= NodeLeastVersion
}

func (p *EnvProvisioner) promptNodeInstall() error {
	fmt.Println()
	color.Yellow("⚠️ Node.js is not installed or not found in PATH.")
	color.Cyan("🔧 Node.js is required to run the agent.")
	fmt.Println()

	options := []string{
		"🚀 Install automatically (recommended)",
		"📖 Exit and show manual installation guide",
	}

	var ans string
	prompt := &survey.Select{
		Message: "How would you like to install Node.js?",
		Options: options,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		return fmt.Errorf("selection error: %w", err)
	}

	switch ans {
	case options[0]:
		fmt.Println()
		color.Green("🚀 Installing Node.js automatically...")

		if err := p.installNodeAutomatically(); err != nil {
			color.Red("❌ Installation failed: %v", err)
			fmt.Println()
			p.showNodeManualInstallation()
			return errors.New("node.js installation failed")
		}

		color.Green("✅ Node.js installation completed!")
		fmt.Println()
		color.Blue("🔍 Verifying installation...")

		if p.checkNodeInstall() {
			color.Green("🎉 Node.js is now available!")
			return nil
		} else {
			color.Yellow("⚠️ Node.js installation completed but not found in PATH.")
			color.Cyan("💡 You may need to restart your terminal or source your shell profile.")
			return errors.New("node.js installed but not in PATH")
		}

	case options[1]:
		p.showNodeManualInstallation()
		return errors.New("node.js not installed")

	default:
		return errors.New("invalid selection")
	}
}

func (p *EnvProvisioner) installNodeAutomatically() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %w", err)
	}

	fnmBinPath := filepath.Join(homeDir, ".local/share/fnm/fnm")
	if runtime.GOOS == "windows" {
		fnmBinPath = filepath.Join(homeDir, "AppData/Roaming/fnm/fnm.exe")
	}

	switch runtime.GOOS {
	case "windows":
		color.Cyan("📦 For Windows, we recommend installing fnm via: 'winget install Schniz.fnm'")
		return errors.New("automatic fnm installation on Windows is not implemented in this script")

	case "darwin", "linux":
		color.Cyan("🚀 Installing fnm (Fast Node Manager)...")
		installFnmCmd := exec.Command("bash", "-c", "curl -fsSL https://fnm.vercel.app/install | bash -s -- --skip-shell")
		installFnmCmd.Stdout = os.Stdout
		installFnmCmd.Stderr = os.Stderr
		if err := installFnmCmd.Run(); err != nil {
			return fmt.Errorf("failed to install fnm: %w", err)
		}

		if _, err := os.Stat(fnmBinPath); os.IsNotExist(err) {
			path, err := exec.LookPath("fnm")
			if err == nil {
				fnmBinPath = path
			} else {
				return errors.New("fnm was installed but binary not found at " + fnmBinPath)
			}
		}

		color.Cyan("📦 Installing Node.js via fnm...")
		installNodeCmd := exec.Command(fnmBinPath, "install", "--lts")
		installNodeCmd.Stdout = os.Stdout
		installNodeCmd.Stderr = os.Stderr
		if err := installNodeCmd.Run(); err != nil {
			return fmt.Errorf("failed to install node via fnm: %w", err)
		}

		color.Cyan("✅ Setting LTS as default Node.js version...")
		useNodeCmd := exec.Command(fnmBinPath, "default", "lts-latest")
		return useNodeCmd.Run()

	default:
		return errors.New("unsupported OS for automatic installation")
	}
}

func (p *EnvProvisioner) showNodeManualInstallation() {
	fmt.Println()

	color.New(color.FgGreen, color.Bold).Println("📖 Manual Node.js Installation Guide")
	fmt.Println()

	fmt.Println(color.MagentaString("Choose one of the following installation methods:"))
	fmt.Println()

	color.Cyan("Method 1: Install via package manager")
	color.Cyan("macOS (brew): brew install node")
	color.Cyan("Ubuntu/Debian: sudo apt install -y nodejs npm")
	color.Cyan("Windows: download from https://nodejs.org and run installer")
	fmt.Println()

	color.Yellow("Method 2: Download from official website")
	color.Yellow("1. Download Node.js from https://nodejs.org/en/download/")
	color.Yellow("2. Follow installer instructions and add to PATH if needed")
	fmt.Println()

	color.Green("✅ Verify Installation")
	fmt.Println(color.WhiteString("node -v"))
	fmt.Println(color.WhiteString("npm -v"))
	fmt.Println()

	color.Cyan("💡 After installation, restart your terminal or source your shell profile.")
	fmt.Println()
}

func (p *EnvProvisioner) checkAgentInstall() bool {
	cmd := exec.Command(string(p.core), "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (p *EnvProvisioner) promptAgentInstall() error {
	fmt.Println()
	color.Yellow("⚠️ %s is not installed or not found in PATH.", p.core)
	color.Cyan("🔧 %s is required to run the agent.", p.core)
	fmt.Println()

	options := []string{
		"🚀 Install automatically",
		"📖 Exit and show manual installation guide",
	}

	var ans string
	prompt := &survey.Select{
		Message: "How would you like to install " + string(p.core) + "?",
		Options: options,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		return fmt.Errorf("selection error: %w", err)
	}

	switch ans {
	case options[0]:
		fmt.Println()
		color.Green("🚀 Installing %s automatically...", p.core)

		if err := p.installAgentAutomatically(); err != nil {
			color.Red("❌ Installation failed: %v", err)
			fmt.Println()
			p.showAgentManualInstallation()
			return errors.New(string(p.core) + " installation failed")
		}
		fmt.Println()
		color.Blue("🔍 Verifying installation...")

		if p.checkAgentInstall() {
			color.Green("🎉 %s is now available!", p.core)
			return nil
		} else {
			color.Yellow("⚠️ %s installed but not found in PATH.", p.core)
			color.Cyan("💡 You may need to restart your terminal or source your shell profile.")
			return errors.New(string(p.core) + " installed but not in PATH")
		}

	case options[1]:
		p.showAgentManualInstallation()
		return errors.New(string(p.core) + " not installed")

	default:
		return errors.New("invalid selection")
	}
}

func (p *EnvProvisioner) installAgentAutomatically() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/C", p.installCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "darwin":
		cmd := exec.Command("bash", "-c", p.installCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "linux":
		cmd := exec.Command("bash", "-c", p.installCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return errors.New("unsupported OS for automatic installation")
	}
}

func (p *EnvProvisioner) showAgentManualInstallation() {
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("📖 Manual %s Installation Guide\n", p.core)
	fmt.Println()

	color.Cyan(fmt.Sprintf("1. Go to official release page: %s", p.releasePage))
	fmt.Printf(color.CyanString("2. Download %s for your OS\n"), p.core)
	color.Cyan("3. Make it executable and place it in a directory in your PATH")

	fmt.Println()
	color.Cyan("💡 After installation, restart your terminal or source your shell profile.")
	fmt.Println()
}
