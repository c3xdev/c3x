package config

import (
	"fmt"

	"github.com/c3xdev/c3x/internal/domain"
)

// Resolved is the fully-merged configuration that the rest of the engine
// reads from. Every field is owned: callers do not retain a viper.Viper
// reference. Downstream modules import only this struct.
type Resolved struct {
	// Region is the default cloud region used when the IaC input doesn't
	// declare one (e.g. provider "aws" without a region attribute).
	Region string

	// Currency in which costs are reported. Defaults to USD.
	Currency domain.Currency

	// Format is the renderer output format: "text", "markdown", "json".
	Format string

	// PricingEndpoint is the GraphQL endpoint URL of the pricing API.
	PricingEndpoint string

	// PricingToken authenticates requests to the pricing API. When
	// non-empty it is sent as `Authorization: Bearer <token>` — the
	// scheme a self-hosted c3x-pricing-api accepts when its API_KEY is
	// set (it also honours X-Api-Key). Empty means no auth header,
	// matching the public pricing.c3x.dev endpoint. Source it from the
	// C3X_PRICING_TOKEN env var or `pricing.token` in config; never
	// commit it.
	PricingToken string

	// Offline forces the calculator to use the offline stub price source.
	Offline bool

	// NoRemoteModules disables network module fetching (Terraform Registry,
	// Git, HTTP archives) while keeping live pricing. Set it when parsing
	// UNTRUSTED input (e.g. user-uploaded Terraform on a server): a crafted
	// `module { source = ... }` would otherwise cause the fetcher to shell
	// out to git or hit an arbitrary URL — an SSRF / metadata-exfiltration
	// vector. Distinct from Offline, which also stubs pricing.
	NoRemoteModules bool

	// AllowFileFunctions enables file(), templatefile(), fileset() and the
	// other filesystem functions, reading only inside the project
	// directory. Off by default: on a pull request from a fork the parsed
	// configuration is attacker-controlled, and a file read, even one
	// confined to the checkout, can leak a credential written there by an
	// earlier CI step through the costs posted back to the PR.
	//
	// It is taken only from sources the runner controls — the flag,
	// C3X_ALLOW_FILE_FUNCTIONS, or the user config file — never from the
	// project's .c3x.toml, which a pull request can edit. NoRemoteModules
	// (untrusted-input mode) forces it off regardless.
	AllowFileFunctions bool

	// NoCache disables the on-disk SQLite cache for this run.
	NoCache bool

	// CachePath is the SQLite file used by the disk cache. Empty string
	// means "use the platform default" (XDG cache dir).
	CachePath string

	// UsagePath is the path to a `c3x-usage.yml` file with runtime usage
	// quantities. Empty means no usage file.
	UsagePath string

	// ResourcesPath overrides the embedded TOML catalog with an
	// on-disk directory. Empty means use the embedded one.
	ResourcesPath string

	// Budget, when > 0, fails the run if the project total exceeds it.
	// Used by `c3x estimate --budget`.
	Budget float64

	// BudgetDelta, when > 0, fails `c3x diff` if the delta exceeds it.
	BudgetDelta float64

	// Verbosity is the count of -v CLI flags (0..3).
	Verbosity int
}

// Defaults returns a Resolved populated with the c3x defaults. This is
// the bottom layer of the 5-layer precedence stack.
func Defaults() Resolved {
	return Resolved{
		Region:          "",
		Currency:        domain.CurrencyUSD,
		Format:          "text",
		PricingEndpoint: "https://pricing.c3x.dev/graphql",
		PricingToken:    "",
		Offline:         false,
		NoCache:         false,
		CachePath:       "",
		UsagePath:       "",
		ResourcesPath:   "",
		Budget:          0,
		BudgetDelta:     0,
		Verbosity:       0,
	}
}

// Validate sanity-checks the merged config. Called once after Resolve
// so downstream code can trust the values.
func (r Resolved) Validate() error {
	// Keep this list in lockstep with render.ParseFormat (config can't
	// import render — it would invert the dependency direction — so
	// the verifier test in config_test pins the two lists together).
	switch r.Format {
	case "text", "markdown", "md", "json", "junit", "html", "csv", "sarif":
	default:
		return fmt.Errorf("unsupported format %q (want text|markdown|json|junit|html|csv|sarif)", r.Format)
	}
	if r.Currency == domain.CurrencyUnknown {
		return fmt.Errorf("currency is unset (config layer didn't supply a default)")
	}
	if r.Budget < 0 {
		return fmt.Errorf("budget must be non-negative, got %v", r.Budget)
	}
	if r.PricingEndpoint == "" && !r.Offline {
		return fmt.Errorf("pricing endpoint is empty and offline mode is not set")
	}
	return nil
}
