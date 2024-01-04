// Copyright (c) 2022 Alibaba Group Holding Ltd.
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

package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/alibaba/higress/pkg/bootstrap"
	innerconstants "github.com/alibaba/higress/pkg/config/constants"
	"github.com/spf13/cobra"
	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pkg/cmd"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/keepalive"
	"istio.io/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DefaultIp   = "127.0.0.1"
	DefaultPort = ":15051"
)

var (
	serverArgs     *bootstrap.ServerArgs
	loggingOptions = log.DefaultOptions()

	serverProvider = func(args *bootstrap.ServerArgs) (bootstrap.ServerInterface, error) {
		return bootstrap.NewServer(args)
	}

	waitForMonitorSignal = func(stop chan struct{}) {
		cmd.WaitSignal(stop)
	}
)

// getServerCommand returns the server cobra command to be executed.
func getServerCommand() *cobra.Command {
	serveCmd := &cobra.Command{
		Use:     "serve",
		Aliases: []string{"serve"},
		Short:   "Starts the higress ingress controller",
		Example: "higress serve",
		PreRunE: func(c *cobra.Command, args []string) error {
			return log.Configure(loggingOptions)
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())

			stop := make(chan struct{})

			server, err := serverProvider(serverArgs)
			if err != nil {
				return fmt.Errorf("fail to create higress server: %v", err)
			}

			if err := server.Start(stop); err != nil {
				return fmt.Errorf("fail to start higress server: %v", err)
			}

			waitForMonitorSignal(stop)

			server.WaitUntilCompletion()
			return nil
		},
	}

	serverArgs = &bootstrap.ServerArgs{
		Debug:                true,
		NativeIstio:          true,
		HttpAddress:          ":8888",
		GrpcAddress:          ":15051",
		GrpcKeepAliveOptions: keepalive.DefaultOption(),
		XdsOptions: bootstrap.XdsOptions{
			DebounceAfter:     features.DebounceAfter,
			DebounceMax:       features.DebounceMax,
			EnableEDSDebounce: features.EnableEDSDebounce,
		},
	}

	serveCmd.PersistentFlags().StringVar(&serverArgs.GatewaySelectorKey, "gatewaySelectorKey", "higress", "gateway resource selector label key")
	serveCmd.PersistentFlags().StringVar(&serverArgs.GatewaySelectorValue, "gatewaySelectorValue", "higress-gateway", "gateway resource selector label value")
	serveCmd.PersistentFlags().BoolVar(&serverArgs.EnableStatus, "enableStatus", true, "enable the ingress status syncer which use to update the ip in ingress's status")
	serveCmd.PersistentFlags().StringVar(&serverArgs.IngressClass, "ingressClass", innerconstants.DefaultIngressClass, "if not empty, only watch the ingresses have the specified class, otherwise watch all ingresses")
	serveCmd.PersistentFlags().StringVar(&serverArgs.WatchNamespace, "watchNamespace", "", "if not empty, only wath the ingresses in the specified namespace, otherwise watch in all namespacees")
	serveCmd.PersistentFlags().BoolVar(&serverArgs.Debug, "debug", serverArgs.Debug, "if true, enables more debug http api")
	serveCmd.PersistentFlags().StringVar(&serverArgs.HttpAddress, "httpAddress", serverArgs.HttpAddress, "the http address")
	serveCmd.PersistentFlags().StringVar(&serverArgs.GrpcAddress, "grpcAddress", serverArgs.GrpcAddress, "the grpc address")
	serveCmd.PersistentFlags().BoolVar(&serverArgs.KeepStaleWhenEmpty, "keepStaleWhenEmpty", false, "keep the stale service entry when there are no endpoints in the service")
	serveCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.ClusterRegistriesNamespace, "clusterRegistriesNamespace",
		serverArgs.RegistryOptions.ClusterRegistriesNamespace, "Namespace for ConfigMap which stores clusters configs")
	serveCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeConfig, "kubeconfig", "",
		"Use a Kubernetes configuration file instead of in-cluster configuration")
	// RegistryOptions Controller options
	serveCmd.PersistentFlags().DurationVar(&serverArgs.RegistryOptions.KubeOptions.ResyncPeriod, "resync", 60*time.Second,
		"Controller resync interval")
	serveCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeOptions.DomainSuffix, "domain", constants.DefaultKubernetesDomain,
		"DNS domain suffix")
	serveCmd.PersistentFlags().StringVar((*string)(&serverArgs.RegistryOptions.KubeOptions.ClusterID), "clusterID", "Kubernetes",
		"The ID of the cluster that this instance resides")
	serveCmd.PersistentFlags().StringToStringVar(&serverArgs.RegistryOptions.KubeOptions.ClusterAliases, "clusterAliases", map[string]string{},
		"Alias names for clusters")
	serveCmd.PersistentFlags().Float32Var(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIQPS, "kubernetesApiQPS", 80.0,
		"Maximum QPS when communicating with the kubernetes API")

	serveCmd.PersistentFlags().IntVar(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIBurst, "kubernetesApiBurst", 160,
		"Maximum burst for throttle when communicating with the kubernetes API")

	loggingOptions.AttachCobraFlags(serveCmd)
	serverArgs.GrpcKeepAliveOptions.AttachCobraFlags(serveCmd)

	return serveCmd
}

// getDebugServerCommand returns the debug server cobra command to be executed.
// We can use this command to debug the higress core container.
func getDebugServerCommand() *cobra.Command {
	debugCmd := &cobra.Command{
		Use:     "debug",
		Aliases: []string{"debug"},
		Short:   "Starts the higress ingress controller in debug mode",
		Example: "higress debug",
		PreRunE: func(c *cobra.Command, args []string) error {
			return log.Configure(loggingOptions)
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())

			// get local no loopback ip
			ip, err := getNonLoopbackIPv4()
			if err != nil {
				return fmt.Errorf("fail to get local no loopback ip: %v", err)
			}

			// get kubernetes clientSet
			kubeConfig := serverArgs.RegistryOptions.KubeConfig
			config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
			if err != nil {
				return fmt.Errorf("fail to build config from kubeconfig: %v", err)
			}
			clientSet, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("fail to create kubernetes clientset: %v", err)
			}

			// update xds address in higress-config ConfigMap
			// and trigger rollout for higress-controller and higress-gateway deployments
			err = updateXdsIpAndRollout(clientSet, ip, serverArgs.GrpcAddress)
			if err != nil {
				return fmt.Errorf("fail to update xds address in higress-config ConfigMap: %v", err)
			}

			stop := make(chan struct{})

			server, err := serverProvider(serverArgs)
			if err != nil {
				return fmt.Errorf("fail to create higress server: %v", err)
			}

			if err := server.Start(stop); err != nil {
				return fmt.Errorf("fail to start higress server: %v", err)
			}

			waitForMonitorSignal(stop)

			server.WaitUntilCompletion()
			return nil
		},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("fail to get user home dir: %v", err)
		os.Exit(1)
	}
	kubeConfigDir := homeDir + "/.kube/config"

	serverArgs = &bootstrap.ServerArgs{
		Debug:                true,
		NativeIstio:          true,
		HttpAddress:          ":8888",
		GrpcAddress:          ":15051",
		GrpcKeepAliveOptions: keepalive.DefaultOption(),
		XdsOptions: bootstrap.XdsOptions{
			DebounceAfter:     features.DebounceAfter,
			DebounceMax:       features.DebounceMax,
			EnableEDSDebounce: features.EnableEDSDebounce,
		},
	}

	debugCmd.PersistentFlags().StringVar(&serverArgs.GatewaySelectorKey, "gatewaySelectorKey", "higress", "gateway resource selector label key")
	debugCmd.PersistentFlags().StringVar(&serverArgs.GatewaySelectorValue, "gatewaySelectorValue", "higress-system-higress-gateway", "gateway resource selector label value")
	debugCmd.PersistentFlags().BoolVar(&serverArgs.EnableStatus, "enableStatus", true, "enable the ingress status syncer which use to update the ip in ingress's status")
	debugCmd.PersistentFlags().StringVar(&serverArgs.IngressClass, "ingressClass", innerconstants.DefaultIngressClass, "if not empty, only watch the ingresses have the specified class, otherwise watch all ingresses")
	debugCmd.PersistentFlags().StringVar(&serverArgs.WatchNamespace, "watchNamespace", "", "if not empty, only wath the ingresses in the specified namespace, otherwise watch in all namespacees")
	debugCmd.PersistentFlags().BoolVar(&serverArgs.Debug, "debug", serverArgs.Debug, "if true, enables more debug http api")
	debugCmd.PersistentFlags().StringVar(&serverArgs.HttpAddress, "httpAddress", serverArgs.HttpAddress, "the http address")
	debugCmd.PersistentFlags().StringVar(&serverArgs.GrpcAddress, "grpcAddress", serverArgs.GrpcAddress, "the grpc address")
	debugCmd.PersistentFlags().BoolVar(&serverArgs.KeepStaleWhenEmpty, "keepStaleWhenEmpty", false, "keep the stale service entry when there are no endpoints in the service")
	debugCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.ClusterRegistriesNamespace, "clusterRegistriesNamespace",
		serverArgs.RegistryOptions.ClusterRegistriesNamespace, "Namespace for ConfigMap which stores clusters configs")
	debugCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeConfig, "kubeconfig", kubeConfigDir,
		"Use a Kubernetes configuration file instead of in-cluster configuration")
	// RegistryOptions Controller options
	debugCmd.PersistentFlags().DurationVar(&serverArgs.RegistryOptions.KubeOptions.ResyncPeriod, "resync", 60*time.Second,
		"Controller resync interval")
	debugCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeOptions.DomainSuffix, "domain", constants.DefaultKubernetesDomain,
		"DNS domain suffix")
	debugCmd.PersistentFlags().StringVar((*string)(&serverArgs.RegistryOptions.KubeOptions.ClusterID), "clusterID", "Kubernetes",
		"The ID of the cluster that this instance resides")
	debugCmd.PersistentFlags().StringToStringVar(&serverArgs.RegistryOptions.KubeOptions.ClusterAliases, "clusterAliases", map[string]string{},
		"Alias names for clusters")
	debugCmd.PersistentFlags().Float32Var(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIQPS, "kubernetesApiQPS", 80.0,
		"Maximum QPS when communicating with the kubernetes API")

	debugCmd.PersistentFlags().IntVar(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIBurst, "kubernetesApiBurst", 160,
		"Maximum burst for throttle when communicating with the kubernetes API")

	loggingOptions.AttachCobraFlags(debugCmd)
	serverArgs.GrpcKeepAliveOptions.AttachCobraFlags(debugCmd)

	return debugCmd
}

func getRecoverCmd() *cobra.Command {
	recoverCmd := &cobra.Command{
		Use:     "recover",
		Aliases: []string{"recover"},
		Short:   "Recover the xds address in higress-config ConfigMap",
		Example: "higress recover",
		RunE: func(c *cobra.Command, args []string) error {
			// get kubernetes clientSet
			kubeConfig := serverArgs.RegistryOptions.KubeConfig
			config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
			if err != nil {
				return fmt.Errorf("fail to build config from kubeconfig: %v", err)
			}
			clientSet, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("fail to create kubernetes clientset: %v", err)
			}

			// recover the xds address in higress-config ConfigMap
			// and trigger rollout for higress-controller and higress-gateway deployments
			// TODO: save ip and port before debug?
			err = updateXdsIpAndRollout(clientSet, DefaultIp, DefaultPort)
			if err != nil {
				return fmt.Errorf("fail to update xds address in higress-config ConfigMap: %v", err)
			}
			return nil
		},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("fail to get user home dir: %v", err)
		os.Exit(1)
	}
	kubeConfigDir := homeDir + "/.kube/config"
	recoverCmd.PersistentFlags().StringVar(&kubeConfigDir, "kubeconfig", kubeConfigDir,
		"Use a Kubernetes configuration file instead of in-cluster configuration")

	return recoverCmd
}

// getNonLoopbackIPv4 returns the first non-loopback IPv4 address of the host.
func getNonLoopbackIPv4() (string, error) {
	// get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// traverse all network interfaces
	for _, i := range interfaces {
		// exclude loopback interface and virtual interface
		if i.Flags&net.FlagLoopback == 0 && i.Flags&net.FlagUp != 0 {
			// get all addresses of the interface
			addrs, err := i.Addrs()
			if err != nil {
				return "", err
			}

			// traverse all addresses of the interface
			for _, addr := range addrs {
				// check the type of the address is IP address
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					// check the IP address is IPv4 address
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String(), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("Non-loopback IPv4 address not found")
}

// updateXdsIpAndRollout updates the xds address in higress-config ConfigMap
// and triggers rollout for higress-controller and higress-gateway deployments
// also can recover the xds address in higress-config ConfigMap
func updateXdsIpAndRollout(c *kubernetes.Clientset, ip string, port string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get higress-config ConfigMap
	cm, err := c.CoreV1().ConfigMaps("higress-system").Get(ctx, "higress-config", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Update mesh field in higress-config ConfigMap
	if _, ok := cm.Data["mesh"]; !ok {
		return fmt.Errorf("mesh not found in configmap higress-config")
	}
	mesh := cm.Data["mesh"]
	newMesh, err := replaceXDSAddress(mesh, ip, port)
	if err != nil {
		return err
	}
	cm.Data["mesh"] = newMesh

	// Update higress-config ConfigMap
	_, err = c.CoreV1().ConfigMaps("higress-system").Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// Trigger rollout for higress-controller deployment
	err = triggerRollout(c, "higress-controller")
	if err != nil {
		return err
	}

	// Trigger rollout for higress-gateway deployment
	err = triggerRollout(c, "higress-gateway")
	if err != nil {
		return err
	}

	return nil
}

// triggerRollout triggers rollout for the specified deployment
func triggerRollout(clientset *kubernetes.Clientset, deploymentName string) error {
	deploymentsClient := clientset.AppsV1().Deployments("higress-system")

	// Get the deployment
	deployment, err := deploymentsClient.Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Increment the deployment's revision to trigger a rollout
	deployment.Spec.Template.ObjectMeta.Labels["version"] = time.Now().Format("20060102150405")

	// Update the deployment
	_, err = deploymentsClient.Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// replaceXDSAddress replaces the xds address in the config string with new IP and Port
func replaceXDSAddress(configString, newIP, newPort string) (string, error) {
	// define the regular expression to match xds address
	xdsRegex := regexp.MustCompile(`xds://[0-9.:]+`)

	// find the first match
	match := xdsRegex.FindString(configString)
	if match == "" {
		// if no match, return error
		return "", fmt.Errorf("no xds address found in config string")
	}

	// replace xds address with new IP and Port
	newXDSAddress := fmt.Sprintf("xds://%s%s", newIP, newPort)
	result := xdsRegex.ReplaceAllString(configString, newXDSAddress)

	return result, nil
}
