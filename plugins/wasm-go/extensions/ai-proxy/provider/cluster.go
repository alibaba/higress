package provider

import "fmt"

type plainCluster struct {
	serviceName string
	servicePort int64
	hostName    string
}

func (c plainCluster) ClusterName() string {
	return fmt.Sprintf("outbound|%d||%s", c.servicePort, c.serviceName)
}

func (c plainCluster) HostName() string {
	return c.hostName
}
