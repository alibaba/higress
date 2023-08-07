package client

import (
	"github.com/hudl/fargo"
	"istio.io/pkg/log"
)

type Handler func(application *fargo.Application) error

/*
 * Plan can be used to get the latest information of a service
 *
 *                       (service B)  ┌──────┐
 *                      ┌────────────►│Plan B│
 *                      │   Timer     └──────┘
 *                      │
 * ┌───────────────┐    │
 * │ eureka-server ├────┤
 * └───────────────┘    │
 *                      │
 *                      │
 *                      │(service A)  ┌──────┐
 *                      └────────────►│Plan A│
 *                          Timer     └──────┘
 */

type Plan struct {
	client  EurekaHttpClient
	stop    chan struct{}
	handler Handler
}

func NewPlan(client EurekaHttpClient, serviceName string, handler Handler) *Plan {
	p := &Plan{
		client:  client,
		stop:    make(chan struct{}),
		handler: handler,
	}

	ch := client.ScheduleAppUpdates(serviceName, p.stop)
	go p.watch(ch)
	return p
}

func (p *Plan) Stop() {
	defer close(p.stop)
	p.stop <- struct{}{}
}

func (p *Plan) watch(ch <-chan fargo.AppUpdate) {
	for {
		select {
		case <-p.stop:
			log.Info("stop eureka plan")
			return
		case updateItem := <-ch:
			if updateItem.Err != nil {
				log.Error("get eureka application failed")
				continue
			}
			if err := p.handler(updateItem.App); err != nil {
				log.Error("handle eureka application failed")
			}
		}
	}
}
