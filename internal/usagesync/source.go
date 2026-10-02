// Package usagesync fills a generated usage file from a cloud account's
// own metrics. It holds what is the same for every provider: finding the
// resources to look up, running the lookups with partial-failure
// semantics, and writing the snapshot. Talking to a provider is behind
// [Source], implemented in a subpackage that alone imports that
// provider's SDK; this package never touches a cloud API, a price or a
// cost calculation.
package usagesync

import (
	"context"
	"time"
)

// Window is the period usage is measured over.
type Window struct {
	Start, End time.Time
}

// Target is one resource to look up.
type Target struct {
	// Address is the Terraform address, which is also the key under
	// resource_usage in a usage file.
	Address string
	Kind    string
	// ID is the name the cloud knows the resource by (a bucket's name).
	ID string
	// Region is where the resource lives.
	Region string
}

// Result is what a source measured for one target.
type Result struct {
	// Values are the quantities for the window, keyed by the identifier the
	// catalog's quantity expressions read (standard_storage_gb, ...).
	Values map[string]float64
	// Series holds the same quantities per calendar month ("2026-09"), so
	// a trend can be fitted later.
	Series map[string]map[string]float64
}

// Source measures usage for the resource kinds it supports. Calls must be
// read-only. An error is recorded for that target and does not stop the
// run.
type Source interface {
	// Provider names the provider ("aws").
	Provider() string
	// Emits maps each supported kind to the usage keys it can produce. A
	// test checks every key against the catalog, so a catalog rename fails
	// CI instead of silently syncing nothing.
	Emits() map[string][]string
	// Collect measures one target over the window.
	Collect(ctx context.Context, t Target, w Window) (Result, error)
}
