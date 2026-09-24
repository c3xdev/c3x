package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/spf13/cobra"
)

// errStrictCaveats is returned by --strict when an estimate carries
// caveats. It exits 3, distinct from a budget breach (1), so a pipeline
// can tell "over budget" from "the number itself is not reliable".
var errStrictCaveats = errors.New("strict: estimate has caveats")

const strictHelp = "fail with exit code 3 when any part of the estimate rests on an assumption: a price quoted " +
	"from another region, a lookup that matched nothing, usage not provided, an attribute that could not be evaluated, " +
	"or a stale cached price. The caveats are always shown; this makes them fail the run"

// enforceStrict prints a one-line summary per caveat kind and returns
// errStrictCaveats when strict is set and there is anything to report.
func enforceStrict(cmd *cobra.Command, caveats []domain.LabeledCaveat, strict bool) error {
	if !strict || len(caveats) == 0 {
		return nil
	}
	byCode := map[string]int{}
	for _, c := range caveats {
		byCode[c.Code]++
	}
	codes := make([]string, 0, len(byCode))
	for c := range byCode {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	w := cmd.ErrOrStderr()
	fmt.Fprintf(w, "c3x: --strict: the estimate has %d caveat(s):\n", len(caveats))
	for _, c := range codes {
		fmt.Fprintf(w, "  %-22s %d\n", c, byCode[c])
	}
	return errStrictCaveats
}
