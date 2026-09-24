package calculator

import (
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
)

func TestRegionForUsesTheProvidersOwnDefault(t *testing.T) {
	e := &Engine{defaultRegion: "us-east-1"}
	none := domain.Resource{}
	if got := e.regionFor(none, "azure"); got != "eastus" {
		t.Errorf("azure with an AWS default: got %q, want eastus", got)
	}
	if got := e.regionFor(none, "gcp"); got != "us-central1" {
		t.Errorf("gcp with an AWS default: got %q, want us-central1", got)
	}
	if got := e.regionFor(none, "aws"); got != "us-east-1" {
		t.Errorf("aws: got %q", got)
	}

	e = &Engine{defaultRegion: "westeurope"}
	if got := e.regionFor(none, "azure"); got != "westeurope" {
		t.Errorf("azure with --region westeurope: got %q", got)
	}
	if got := e.regionFor(none, "aws"); got != "us-east-1" {
		t.Errorf("aws with an Azure default: got %q, want us-east-1", got)
	}

	own := "europe-west4"
	if got := e.regionFor(domain.Resource{Region: &own}, "gcp"); got != own {
		t.Errorf("a resource's own region must win: got %q", got)
	}
}
