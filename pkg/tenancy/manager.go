package tenancy

import (
	"fmt"
	"strings"
)

// TenantManager provides namespace-based isolation helpers.
type TenantManager struct{}

// IsolateRoutes validates a namespace name for isolation rules. Real enforcement should be wired in reconcilers.
func (m *TenantManager) IsolateRoutes(namespace string) error {
	if namespace == "" || strings.Contains(namespace, "..") {
		return fmt.Errorf("invalid tenant namespace: %q", namespace)
	}
	return nil
}

// AllowedNamespace returns whether a resource namespace is allowed for a given tenant list.
func (m *TenantManager) AllowedNamespace(resourceNamespace string, tenantNamespaces []string) bool {
	if len(tenantNamespaces) == 0 {
		return true
	}
	for _, ns := range tenantNamespaces {
		if ns == resourceNamespace {
			return true
		}
	}
	return false
}