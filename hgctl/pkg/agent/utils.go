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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/alibaba/higress/hgctl/pkg/helm"
	"github.com/alibaba/higress/hgctl/pkg/installer"
	"github.com/alibaba/higress/hgctl/pkg/kubernetes"
	"github.com/alibaba/higress/v2/pkg/cmd/options"
	"github.com/braydonk/yaml"
	"github.com/fatih/color"
	"github.com/higress-group/openapi-to-mcpserver/pkg/converter"
	"github.com/higress-group/openapi-to-mcpserver/pkg/models"
	"github.com/higress-group/openapi-to-mcpserver/pkg/parser"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var binaryName = AgentBinaryName

// ------ cmd related  ------
func BindFlagToEnv(cmd *cobra.Command, flagName, envName string) {
	_ = viper.BindPFlag(flagName, cmd.PersistentFlags().Lookup(flagName))
	_ = viper.BindEnv(flagName, envName)
}

// ------ Prompt to install prequisite environment  ------
func checkNodeInstall() bool {
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

func promptNodeInstall() error {
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
	case "🚀 Install automatically (recommended)":
		fmt.Println()
		color.Green("🚀 Installing Node.js automatically...")

		if err := installNodeAutomatically(); err != nil {
			color.Red("❌ Installation failed: %v", err)
			fmt.Println()
			showNodeManualInstallation()
			return errors.New("node.js installation failed")
		}

		color.Green("✅ Node.js installation completed!")
		fmt.Println()
		color.Blue("🔍 Verifying installation...")

		if checkNodeInstall() {
			color.Green("🎉 Node.js is now available!")
			return nil
		} else {
			color.Yellow("⚠️ Node.js installation completed but not found in PATH.")
			color.Cyan("💡 You may need to restart your terminal or source your shell profile.")
			return errors.New("node.js installed but not in PATH")
		}

	case "📖 Exit and show manual installation guide":
		showNodeManualInstallation()
		return errors.New("node.js not installed")

	default:
		return errors.New("invalid selection")
	}
}

func installNodeAutomatically() error {
	switch runtime.GOOS {
	case "windows":
		color.Cyan("📦 Please download Node.js installer from https://nodejs.org and run it manually on Windows")
		return errors.New("automatic installation not supported on Windows yet")
	case "darwin":
		// macOS: use brew
		cmd := exec.Command("brew", "install", "node")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "linux":
		// Linux (Debian/Ubuntu example)
		cmd := exec.Command("sudo", "apt", "update")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
		cmd = exec.Command("sudo", "apt", "install", "-y", "nodejs", "npm")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return errors.New("unsupported OS for automatic installation")
	}
}

func showNodeManualInstallation() {
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

func checkAgentInstall() bool {
	cmd := exec.Command(binaryName, "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func promptAgentInstall() error {
	fmt.Println()
	color.Yellow("⚠️ %s is not installed or not found in PATH.", binaryName)
	color.Cyan("🔧 %s is required to run the agent.", binaryName)
	fmt.Println()

	options := []string{
		"🚀 Install automatically (recommended)",
		"📖 Exit and show manual installation guide",
	}

	var ans string
	prompt := &survey.Select{
		Message: "How would you like to install " + binaryName + "?",
		Options: options,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		return fmt.Errorf("selection error: %w", err)
	}

	switch ans {
	case "🚀 Install automatically (recommended)":
		fmt.Println()
		color.Green("🚀 Installing %s automatically...", binaryName)

		if err := installAgentAutomatically(); err != nil {
			color.Red("❌ Installation failed: %v", err)
			fmt.Println()
			showAgentManualInstallation()
			return errors.New(binaryName + " installation failed")
		}

		color.Green("✅ %s installation completed!", binaryName)
		fmt.Println()
		color.Blue("🔍 Verifying installation...")

		if checkAgentInstall() {
			color.Green("🎉 %s is now available!", binaryName)
			return nil
		} else {
			color.Yellow("⚠️ %s installed but not found in PATH.", binaryName)
			color.Cyan("💡 You may need to restart your terminal or source your shell profile.")
			return errors.New(binaryName + " installed but not in PATH")
		}

	case "📖 Exit and show manual installation guide":
		showAgentManualInstallation()
		return errors.New(binaryName + " not installed")

	default:
		return errors.New("invalid selection")
	}
}

func installAgentAutomatically() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/C", AgentInstallCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "darwin":
		cmd := exec.Command("bash", "-c", AgentInstallCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "linux":
		cmd := exec.Command("bash", "-c", AgentInstallCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return errors.New("unsupported OS for automatic installation")
	}
}

func showAgentManualInstallation() {
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("📖 Manual %s Installation Guide\n", binaryName)
	fmt.Println()

	fmt.Println(color.MagentaString("Supported Operating Systems: macOS 10.15+, Ubuntu 20.04+/Debian 10+, or Windows 10+ (WSL/Git for Windows)"))
	fmt.Println(color.MagentaString("Hardware: 4GB+ RAM"))
	fmt.Println(color.MagentaString("Software: Node.js 18+"))
	fmt.Println(color.MagentaString("Network: Internet connection required for authentication and AI processing"))
	fmt.Println(color.MagentaString("Shell: Works best in Bash, Zsh, or Fish"))
	fmt.Println()

	color.Cyan("Method 1: Download prebuilt binary")
	color.Cyan(fmt.Sprintf("1. Go to official release page: %s", AgentReleasePage))
	fmt.Printf(color.CyanString("2. Download %s for your OS\n"), binaryName)
	color.Cyan("3. Make it executable and place it in a directory in your PATH")
	fmt.Println()

	fmt.Println()
	color.Green("✅ Verify Installation")
	fmt.Printf(color.WhiteString("%s --version\n"), binaryName)
	fmt.Println()
	color.Cyan("💡 After installation, restart your terminal or source your shell profile.")
	fmt.Println()
}

// ------ MCP convert utils function  ------
func parseOpenapi2MCP(arg MCPAddArg) *models.MCPConfig {
	path := arg.spec
	serverName := arg.name

	// Create a new parser
	p := parser.NewParser()

	p.SetValidation(true)

	// Parse the OpenAPI specification
	err := p.ParseFile(path)
	if err != nil {
		fmt.Printf("Error parsing OpenAPI specification: %v\n", err)
		os.Exit(1)
	}

	c := converter.NewConverter(p, models.ConvertOptions{
		ServerName:     serverName,
		ToolNamePrefix: "",
		TemplatePath:   "",
	})

	// Convert the OpenAPI specification to an MCP configuration
	config, err := c.Convert()
	if err != nil {
		fmt.Printf("Error converting OpenAPI specification: %v\n", err)
		os.Exit(1)
	}

	return config
}

func convertMCPConfigToStr(cfg *models.MCPConfig) string {
	var data []byte
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)

	if err := encoder.Encode(cfg); err != nil {
		fmt.Printf("Error encoding YAML: %v\n", err)
		os.Exit(1)
	}
	data = buffer.Bytes()
	str := string(data)

	// fmt.Println("Successfully converted OpenAPI specification to MCP Server")
	// fmt.Printf("Get MCP server config string: %v", str)
	return str

	// if err != nil {
	// 	fmt.Printf("Error marshaling MCP configuration: %v\n", err)
	// 	os.Exit(1)
	// }

	// err = os.WriteFile(*outputFile, data, 0644)
	// if err != nil {
	// 	fmt.Printf("Error writing MCP configuration: %v\n", err)
	// 	os.Exit(1)
	// }

}

func GetHigressGatewayServiceIP() (string, error) {
	color.Cyan("🚀 Adding openapi MCP Server to agent, checking Higress Gateway Pod status...")

	defaultKubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", defaultKubeconfig)
	if err != nil {
		color.Yellow("⚠️ Failed to load default kubeconfig: %v", err)
		return promptForServiceKubeSettingsAndRetry()
	}

	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		color.Yellow("⚠️ Failed to create Kubernetes client: %v", err)
		return promptForServiceKubeSettingsAndRetry()
	}

	namespace := "higress-system"
	svc, err := clientset.CoreV1().Services(namespace).Get(context.Background(), "higress-gateway", metav1.GetOptions{})
	if err != nil || svc == nil {
		color.Yellow("⚠️ Could not find Higress Gateway Service in namespace '%s'.", namespace)
		return promptForServiceKubeSettingsAndRetry()
	}

	ip, err := extractServiceIP(clientset, namespace, svc)
	if err != nil {
		return "", err
	}

	color.Green("✅ Found Higress Gateway Service IP: %s (namespace: %s)", ip, namespace)
	return ip, nil
}

// higress-gateway should always be LoadBalancer
func extractServiceIP(clientset *k8s.Clientset, namespace string, svc *v1.Service) (string, error) {
	return svc.Spec.ClusterIP, nil

	// // fallback to Pod IP
	// if len(svc.Spec.Selector) > 0 {
	// 	selector := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: svc.Spec.Selector})
	// 	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
	// 		LabelSelector: selector,
	// 	})
	// 	if err != nil {
	// 		return "", fmt.Errorf("failed to list pods for selector: %v", err)
	// 	}
	// 	if len(pods.Items) > 0 {
	// 		return pods.Items[0].Status.PodIP, nil
	// 	}
	// }

}

// prompt fallback for user input
func promptForServiceKubeSettingsAndRetry() (string, error) {
	color.Cyan("Let's fix it manually 👇")

	kubeconfigPrompt := promptui.Prompt{
		Label:   "Enter kubeconfig path",
		Default: filepath.Join(os.Getenv("HOME"), ".kube", "config"),
	}
	kubeconfigPath, err := kubeconfigPrompt.Run()
	if err != nil {
		return "", fmt.Errorf("aborted: %v", err)
	}

	nsPrompt := promptui.Prompt{
		Label:   "Enter Higress namespace",
		Default: "higress-system",
	}
	namespace, err := nsPrompt.Run()
	if err != nil {
		return "", err
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	svc, err := clientset.CoreV1().Services(namespace).Get(context.Background(), "higress-gateway", metav1.GetOptions{})
	if err != nil || svc == nil {
		color.Red("❌ Higress Gateway Service not found in namespace '%s'", namespace)
		return "", fmt.Errorf("service not found")
	}

	ip, err := extractServiceIP(clientset, namespace, svc)
	if err != nil {
		return "", err
	}

	color.Green("✅ Found Higress Gateway Service IP: %s (namespace: %s)", ip, namespace)
	return ip, nil
}

func getConsoleCredentials(profile *helm.Profile) (username, password string, err error) {
	cliClient, err := kubernetes.NewCLIClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())

	if err != nil {
		return "", "", fmt.Errorf("failed to build kubernetes client: %w", err)
	}

	secret, err := cliClient.KubernetesInterface().CoreV1().Secrets(profile.Global.Namespace).Get(context.Background(), "higress-console", metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	return string(secret.Data["adminUsername"]), string(secret.Data["adminPassword"]), nil
}

// This function will do following things:
// 1. read the profile from local-file
// 2. read the profile from k8s' configMap
// 3. combine the two type profiles together and return
func getAllProfiles() ([]*installer.ProfileContext, error) {
	profileContexts := make([]*installer.ProfileContext, 0)
	profileInstalledPath, err := installer.GetProfileInstalledPath()
	if err != nil {
		return profileContexts, nil
	}
	fileProfileStore, err := installer.NewFileDirProfileStore(profileInstalledPath)
	if err != nil {
		return profileContexts, nil
	}
	fileProfileContexts, err := fileProfileStore.List()
	if err == nil {
		profileContexts = append(profileContexts, fileProfileContexts...)
	}

	cliClient, err := kubernetes.NewCLIClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return profileContexts, nil
	}
	configmapProfileStore, err := installer.NewConfigmapProfileStore(cliClient)
	if err != nil {
		return profileContexts, nil
	}

	configmapProfileContexts, err := configmapProfileStore.List()
	if err == nil {
		profileContexts = append(profileContexts, configmapProfileContexts...)
	}
	return profileContexts, nil
}
