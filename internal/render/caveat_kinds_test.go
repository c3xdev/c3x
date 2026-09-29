package render

import (
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
)

// The summary names the caveats this estimate has, in order of first
// appearance, not a fixed list of examples.
func TestCaveatKindsNamesWhatIsPresent(t *testing.T) {
	c := func(code string) domain.LabeledCaveat {
		return domain.LabeledCaveat{Caveat: domain.Caveat{Code: code}}
	}
	cs := []domain.LabeledCaveat{
		c(domain.CaveatAssumedCount), c(domain.CaveatAssumedCount),
		c(domain.CaveatRegionFallback), c(domain.CaveatAssumedCount),
		c("some_future_code"),
	}
	got := caveatKinds(cs)
	want := "an assumed instance count, a price from another region, some_future_code"
	if got != want {
		t.Errorf("caveatKinds = %q, want %q", got, want)
	}
	for _, code := range []string{
		domain.CaveatRegionFallback, domain.CaveatNoPrice, domain.CaveatUsageMissing,
		domain.CaveatUnresolved, domain.CaveatAssumedCount, domain.CaveatStub, domain.CaveatStalePrice,
	} {
		if _, ok := caveatPhrases[code]; !ok {
			t.Errorf("no plain-words phrase for caveat code %q", code)
		}
	}
}
