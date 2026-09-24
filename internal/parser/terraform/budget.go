package terraform

import (
	"errors"
	"fmt"
	"time"
)

// Limits on a single parse. c3x parses configuration it does not control
// (pull requests from forks, uploads on a server), and without these a few
// lines can exhaust the host: `count = 1000000000` grew past 6 GB before it
// was stopped, and four `module { source = "." }` blocks recursing to
// MaxModuleDepth expand 4^10 times. Real configurations sit orders of
// magnitude below every limit.
const (
	maxInstancesPerResource = 10_000
	maxResourcesPerParse    = 200_000
	maxModuleCallsPerParse  = 5_000
	maxParseDuration        = 2 * time.Minute
)

// errParseLimit is wrapped by every limit error, so callers can tell a
// refused input from a malformed one.
var errParseLimit = errors.New("configuration exceeds c3x's parse limits")

// parseBudget is shared by the root module and every module expanded from
// it, so the limits apply to the whole parse rather than per module.
type parseBudget struct {
	resources   int
	moduleCalls int
	deadline    time.Time
}

func newParseBudget() *parseBudget {
	return &parseBudget{deadline: time.Now().Add(maxParseDuration)}
}

// instances checks one resource's count / for_each size before its
// instances are expanded.
func (b *parseBudget) instances(address string, n int) error {
	if b == nil {
		return nil
	}
	if n > maxInstancesPerResource {
		return fmt.Errorf("%w: %s expands to %d instances (limit %d)", errParseLimit, address, n, maxInstancesPerResource)
	}
	b.resources += n
	if b.resources > maxResourcesPerParse {
		return fmt.Errorf("%w: more than %d resources", errParseLimit, maxResourcesPerParse)
	}
	return b.clock()
}

func (b *parseBudget) moduleCall(name string) error {
	if b == nil {
		return nil
	}
	b.moduleCalls++
	if b.moduleCalls > maxModuleCallsPerParse {
		return fmt.Errorf("%w: more than %d module expansions (at module %q)", errParseLimit, maxModuleCallsPerParse, name)
	}
	return b.clock()
}

func (b *parseBudget) clock() error {
	if b != nil && time.Now().After(b.deadline) {
		return fmt.Errorf("%w: parse took longer than %s", errParseLimit, maxParseDuration)
	}
	return nil
}
