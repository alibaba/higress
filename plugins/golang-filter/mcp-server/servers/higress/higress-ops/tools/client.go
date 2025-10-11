package tools

// OpsClient defines the interface for operations client
type OpsClient interface {
	// GetIstiodDebug calls Istiod debug endpoints
	GetIstiodDebug(path string) ([]byte, error)

	// GetEnvoyAdmin calls Envoy admin endpoints
	GetEnvoyAdmin(path string) ([]byte, error)

	// GetIstiodDebugWithParams calls Istiod debug endpoints with query parameters
	GetIstiodDebugWithParams(path string, params map[string]string) ([]byte, error)

	// GetEnvoyAdminWithParams calls Envoy admin endpoints with query parameters
	GetEnvoyAdminWithParams(path string, params map[string]string) ([]byte, error)

	// GetNamespace returns the configured namespace
	GetNamespace() string

	// GetIstiodURL returns the Istiod URL
	GetIstiodURL() string

	// GetEnvoyAdminURL returns the Envoy admin URL
	GetEnvoyAdminURL() string
}
