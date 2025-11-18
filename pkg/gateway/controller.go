package gateway

import (
	"context"

	"github.com/alibaba/higress/pkg/ingress/kube/gateway/istio"
	"istio.io/istio/pilot/pkg/model"
)

// Controller is a thin adapter around the existing istio gateway controller
// to provide an explicit surface for reconciling Gateway API resources.
// This is primarily useful for testing and modular wiring.
type Controller struct {
	inner *istio.Controller
}

func NewController(inner *istio.Controller) *Controller {
	return &Controller{inner: inner}
}

// Reconcile triggers the inner controller reconciliation for Gateway API
// resources based on the provided push context.
func (c *Controller) Reconcile(ctx context.Context, ps *model.PushContext) error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Reconcile(ps)
}