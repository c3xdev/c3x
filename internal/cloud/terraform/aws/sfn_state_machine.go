package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getStepFunctionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_sfn_state_machine",
		CoreRFunc: NewSFnStateMachine,
	}
}

func NewSFnStateMachine(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.SFnStateMachine{
		Address: d.Address,
		Region:  d.Get("region").String(),
		Type:    d.Get("type").String(),
	}
	return r
}
