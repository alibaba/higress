//go:build higress_integration
// +build higress_integration

package mcptools

// MigrationContext holds the configuration context for migration operations
type MigrationContext struct {
	GatewayName      string
	GatewayNamespace string
	DefaultNamespace string
	DefaultHostname  string
	RoutePrefix      string
	ServicePort      int
	TargetPort       int
}

// NewDefaultMigrationContext creates a MigrationContext with default values
func NewDefaultMigrationContext() *MigrationContext {
	return &MigrationContext{
		GatewayName:      "higress-gateway",
		GatewayNamespace: "higress-system",
		DefaultNamespace: "default",
		DefaultHostname:  "example.com",
		RoutePrefix:      "nginx-migrated",
		ServicePort:      80,
		TargetPort:       8080,
	}
}
