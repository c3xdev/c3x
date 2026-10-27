package aws

import (
	"strings"

	"github.com/c3xdev/c3x/internal/usagesync"
)

// collected is one quantity's outcome. err means that quantity was not
// measured. res may still hold other quantities when several collected
// values are merged.
type collected struct {
	res usagesync.Result
	err error
}

// mergeParts keeps every quantity that was measured. When some were and
// some were not, the failures become Result.Problem and the error is
// nil, so the caller writes the values and records the gap. When none
// were, the errors are returned and no values are.
func mergeParts(parts ...collected) (usagesync.Result, error) {
	var (
		values   map[string]float64
		series   map[string]map[string]float64
		problems []string
		failures []error
		measured bool
	)
	for _, p := range parts {
		if p.err != nil {
			failures = append(failures, p.err)
			continue
		}
		measured = true
		for k, v := range p.res.Values {
			if values == nil {
				values = map[string]float64{}
			}
			values[k] = v
		}
		for k, byMonth := range p.res.Series {
			if len(byMonth) == 0 {
				continue
			}
			if series == nil {
				series = map[string]map[string]float64{}
			}
			series[k] = byMonth
		}
		if p.res.Problem != "" {
			problems = append(problems, p.res.Problem)
		}
	}
	if !measured {
		return usagesync.Result{}, joinFailures(failures)
	}
	for _, err := range failures {
		problems = append(problems, err.Error())
	}
	return usagesync.Result{
		Values:  values,
		Series:  series,
		Problem: strings.Join(problems, "; "),
	}, nil
}

func joinFailures(errs []error) error {
	var out []error
	for _, err := range errs {
		if err != nil {
			out = append(out, err)
		}
	}
	switch len(out) {
	case 0:
		return errNothingMeasured
	case 1:
		return out[0]
	default:
		return joinedError{out}
	}
}

// errNothingMeasured is the fallback when merge is called with nothing.
// The call sites always pass a real failure alongside it.
var errNothingMeasured = errString("nothing measured")

type errString string

func (e errString) Error() string { return string(e) }

// joinedError keeps every cause, joined for the snapshot with "; " so a
// resource's error stays one line. Unwrap lets errors.Is see each cause.
type joinedError struct {
	errs []error
}

func (e joinedError) Error() string {
	parts := make([]string, len(e.errs))
	for i, err := range e.errs {
		parts[i] = err.Error()
	}
	return strings.Join(parts, "; ")
}

func (e joinedError) Unwrap() []error { return e.errs }
