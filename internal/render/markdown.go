package render

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

// RenderMarkdown formats an Estimate as flat, always-expanded
// markdown: an H2 heading, one table per priced resource, and a
// project-total line. This is the lossless baseline used by
// `c3x estimate --format markdown`. The PR/MR comment layout —
// a one-line summary with the tables tucked into a collapsible
// <details> block — is [RenderMarkdownComment].
//
// The output is GitHub-flavored: tables for the line-item breakdown,
// fenced code for the project total, and emoji markers (📦 priced,
// ⚪ free, ⚠️ static-rate) so reviewers can scan visually.
func RenderMarkdown(est domain.Estimate) string {
	var b strings.Builder
	cur := est.Currency
	b.WriteString("## c3x estimate\n\n")
	if len(est.Costs) == 0 {
		b.WriteString("_No resources to estimate._\n")
		return b.String()
	}

	tables, priced := markdownCostTables(est)
	if priced == 0 {
		b.WriteString("_No resources priced (offline mode or unknown kinds)._\n")
		return b.String()
	}
	b.WriteString(tables)
	fmt.Fprintf(&b, "**Project total: %s%s/mo**\n", cur.Symbol(), est.ProjectTotal)
	b.WriteString(caveatsMarkdown(est.AllCaveats()))
	return b.String()
}

// markdownCostTables renders just the per-resource breakdown tables
// (no heading, no total). Shared by the flat [RenderMarkdown] baseline
// and the collapsible [RenderMarkdownComment] so both stay in lockstep.
// Returns the concatenated tables and the count of priced resources.
func markdownCostTables(est domain.Estimate) (string, int) {
	var b strings.Builder
	cur := est.Currency
	priced := 0
	for _, c := range est.Costs {
		if len(c.LineItems) == 0 {
			continue
		}
		priced++
		marker := "📦"
		if c.HasStaticRate() {
			marker = "⚠️"
		}
		fmt.Fprintf(&b, "### %s `%s`  —  %s%s/mo\n\n", marker, c.Resource.Label(),
			cur.Symbol(), c.MonthlySubtotal)
		b.WriteString("| Dimension | Quantity | Unit rate | Monthly | Source |\n")
		b.WriteString("|---|---:|---:|---:|---|\n")
		for _, li := range c.LineItems {
			fmt.Fprintf(&b, "| %s | %s %s | %s%s | %s%s | %s |\n",
				escapeMD(li.Description),
				li.Quantity, li.Unit,
				cur.Symbol(), li.UnitRate,
				cur.Symbol(), li.MonthlyCost,
				li.PriceSource)
		}
		b.WriteString("\n")
	}
	return b.String(), priced
}

// RenderMarkdownComment is the layout `c3x comment <forge>` posts to a
// PR/MR: a one-line cost summary with the per-resource breakdown tucked
// into a collapsible <details> block. On a busy MR with many resources
// the headline cost stays visible while the tables are one click away —
// restoring the summary+details layout of the pre-relaunch CLI.
//
// The blank lines around the tables inside <details> are required for
// GitHub/GitLab to render the markdown within the collapsed section.
func RenderMarkdownComment(est domain.Estimate) string {
	cur := est.Currency
	var b strings.Builder
	b.WriteString("#### 💰 C3X report\n\n")

	if len(est.Costs) == 0 {
		b.WriteString("_No resources to estimate._\n")
		return b.String()
	}
	tables, priced := markdownCostTables(est)
	if priced == 0 {
		b.WriteString("_No resources priced (offline mode or unknown kinds)._\n")
		return b.String()
	}

	fmt.Fprintf(&b, "**Monthly estimate: %s%s/mo**\n\n", cur.Symbol(), est.ProjectTotal)
	b.WriteString("<details><summary>Estimate details</summary>\n\n")
	b.WriteString(tables)
	b.WriteString("</details>\n")
	b.WriteString(caveatsMarkdown(est.AllCaveats()))
	return b.String()
}

// RenderMarkdownDiff formats a Diff for PR comments. Resources are
// grouped by change kind so reviewers see Added/Removed/Modified
// sections rather than a flat list. The Δ column carries an
// up/down/flat indicator — at a glance reviewers see which way the
// bill moves without parsing signed numbers.
func RenderMarkdownDiff(d domain.Diff) string {
	var b strings.Builder
	cur := d.Currency
	b.WriteString("## c3x diff\n\n")
	fmt.Fprintf(&b, "**Total: %s%s/mo → %s%s/mo  %s**\n\n",
		cur.Symbol(), d.BaselineTotal,
		cur.Symbol(), d.CurrentTotal,
		signedWithIndicator(cur.Symbol(), d.TotalDelta.String()))
	b.WriteString(markdownDiffGroups(d))
	b.WriteString(caveatsMarkdown(d.Caveats))
	return b.String()
}

// markdownDiffGroups renders the Added/Modified/Removed sections of a
// Diff (no heading, no total line). Shared by the flat
// [RenderMarkdownDiff] baseline and the collapsible
// [RenderMarkdownDiffComment].
func markdownDiffGroups(d domain.Diff) string {
	var b strings.Builder
	cur := d.Currency
	groups := map[domain.DeltaKind][]domain.ResourceDelta{}
	for _, r := range d.Resources {
		groups[r.Kind] = append(groups[r.Kind], r)
	}
	for _, kind := range []struct {
		k     domain.DeltaKind
		emoji string
		label string
	}{
		{domain.DeltaAdded, "🟢", "Added"},
		{domain.DeltaModified, "🟡", "Modified"},
		{domain.DeltaRemoved, "🔴", "Removed"},
	} {
		rs := groups[kind.k]
		if len(rs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s %s\n\n", kind.emoji, kind.label)
		b.WriteString("| Resource | Baseline | Current | Δ |\n")
		b.WriteString("|---|---:|---:|---:|\n")
		for _, r := range rs {
			fmt.Fprintf(&b, "| `%s` | %s%s | %s%s | %s |\n",
				r.Resource.Label(),
				cur.Symbol(), r.Baseline,
				cur.Symbol(), r.Current,
				signedWithIndicator(cur.Symbol(), r.Delta.String()))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// RenderMarkdownDiffComment is the collapsible PR/MR layout for a cost
// diff. The headline leads with the dollar change ("Monthly cost
// increased by $144.00 (+16.1%)") because that delta is what financial
// approvers act on, followed by a baseline/new/change table; the
// per-resource Added/Modified/Removed breakdown collapses into a
// <details> block, one click away.
func RenderMarkdownDiffComment(d domain.Diff) string {
	sym := d.Currency.Symbol()
	var b strings.Builder
	b.WriteString("#### 💰 C3X report\n\n")

	// Lead with the change in dollars (and percent) — the number
	// approvers care about most, mirroring the pre-rewrite CLI.
	pct := diffPercent(d)
	switch {
	case d.TotalDelta.IsPositive():
		fmt.Fprintf(&b, "**Monthly cost increased by %s%s%s 📈**\n\n",
			sym, d.TotalDelta.Abs().StringFixed(2), pct)
	case d.TotalDelta.IsNegative():
		fmt.Fprintf(&b, "**Monthly cost decreased by %s%s%s 📉**\n\n",
			sym, d.TotalDelta.Abs().StringFixed(2), pct)
	default:
		fmt.Fprintf(&b, "**Monthly cost unchanged at %s%s/mo**\n\n",
			sym, d.CurrentTotal.StringFixed(2))
	}

	// Baseline / new / change so all three numbers are visible at once.
	b.WriteString("| Baseline | New | Change |\n|---:|---:|---:|\n")
	fmt.Fprintf(&b, "| %s%s/mo | %s%s/mo | %s |\n\n",
		sym, d.BaselineTotal.StringFixed(2),
		sym, d.CurrentTotal.StringFixed(2),
		signedWithIndicator(sym, d.TotalDelta.StringFixed(2)))

	groups := markdownDiffGroups(d)
	if groups == "" {
		b.WriteString("_No line-item changes._\n")
	} else {
		b.WriteString("<details><summary>Estimate details</summary>\n\n")
		b.WriteString(groups)
		b.WriteString("</details>\n")
	}
	b.WriteString(caveatsMarkdown(d.Caveats))
	return b.String()
}

// diffPercent renders the change as a percent of the baseline —
// " (+16.1%)" / " (-5.2%)" — for the diff-comment headline. Returns ""
// when there's no baseline to divide by (a brand-new project) or the
// change is zero, so the caller can omit it cleanly.
func diffPercent(d domain.Diff) string {
	if d.BaselineTotal.IsZero() || d.TotalDelta.IsZero() {
		return ""
	}
	pct := d.TotalDelta.Div(d.BaselineTotal).Mul(decimal.NewFromInt(100))
	sign := "+"
	if pct.IsNegative() {
		sign = "-"
	}
	return fmt.Sprintf(" (%s%s%%)", sign, pct.Abs().StringFixed(1))
}

// escapeMD escapes the pipe character so resource labels and
// descriptions don't break the table layout. We don't escape `_` or
// `*` because GitHub renders them harmlessly inside table cells.
func escapeMD(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

// caveatsMarkdown is the caveat section appended to every markdown
// layout, PR comments included: one visible line saying the total rests
// on assumptions, and the list behind a <details> so a long one doesn't
// flood the thread. Empty when there is nothing to qualify.
func caveatsMarkdown(cs []domain.LabeledCaveat) string {
	if len(cs) == 0 {
		return ""
	}
	var b strings.Builder
	noun := "caveats"
	if len(cs) == 1 {
		noun = "caveat"
	}
	fmt.Fprintf(&b, "\n> ⚠️ **%d %s:** parts of this estimate rest on assumptions (a price from another region, usage not provided, an attribute that could not be evaluated), so it may misstate the real cost.\n\n", len(cs), noun)
	b.WriteString("<details><summary>Caveats</summary>\n\n| Resource | Line | Caveat |\n|---|---|---|\n")
	for _, c := range cs {
		line := c.Line
		if line == "" {
			line = "—"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", c.Resource.Label(), escapeCell(line), escapeCell(c.Detail))
	}
	b.WriteString("\n</details>\n")
	return b.String()
}

func escapeCell(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
