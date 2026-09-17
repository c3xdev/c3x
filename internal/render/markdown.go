package render

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
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
// diff: the headline total-delta line stays visible while the
// Added/Modified/Removed tables collapse into a <details> block. The
// summary is the per-PR delta ("$894/mo → $1,038/mo ▲ +$144") reviewers
// care about; the breakdown is one click away.
func RenderMarkdownDiffComment(d domain.Diff) string {
	cur := d.Currency
	var b strings.Builder
	b.WriteString("#### 💰 C3X report\n\n")
	fmt.Fprintf(&b, "**Monthly estimate: %s%s/mo → %s%s/mo  %s**\n\n",
		cur.Symbol(), d.BaselineTotal,
		cur.Symbol(), d.CurrentTotal,
		signedWithIndicator(cur.Symbol(), d.TotalDelta.String()))

	groups := markdownDiffGroups(d)
	if groups == "" {
		b.WriteString("_No cost changes in this change set._\n")
		return b.String()
	}
	b.WriteString("<details><summary>Estimate details</summary>\n\n")
	b.WriteString(groups)
	b.WriteString("</details>\n")
	return b.String()
}

// escapeMD escapes the pipe character so resource labels and
// descriptions don't break the table layout. We don't escape `_` or
// `*` because GitHub renders them harmlessly inside table cells.
func escapeMD(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}
