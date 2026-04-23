package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRoute53HealthCheck() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_route53_health_check",
		CoreRFunc: NewRoute53HealthCheck,
	}
}

func NewRoute53HealthCheck(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.Route53HealthCheck{
		Address:         d.Address,
		Type:            d.Get("type").String(),
		RequestInterval: d.Get("request_interval").String(),
		MeasureLatency:  d.Get("measure_latency").Bool(),
	}
	return r
}
