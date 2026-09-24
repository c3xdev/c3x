package render

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

// RenderJSON marshals an Estimate to indented JSON. The serialised
// shape is the public contract for `c3x estimate --format json`; any
// future schema change to domain.Estimate's exported fields is a
// breaking change.
func RenderJSON(est domain.Estimate) (string, error) {
	view := estimateView{
		Costs:        toCostViews(est.Costs),
		ProjectTotal: est.ProjectTotal.String(),
		Currency:     est.Currency.String(),
		GeneratedAt:  est.GeneratedAt.UTC().Format("2006-01-02T15:04:05Z"),
		CaveatCount:  est.CaveatCount(),
	}
	b, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal estimate: %w", err)
	}
	return string(b) + "\n", nil
}

// RenderJSONDiff marshals a Diff to indented JSON. Same versioning
// contract as RenderJSON.
func RenderJSONDiff(d domain.Diff) (string, error) {
	view := diffView{
		BaselineTotal: d.BaselineTotal.String(),
		CurrentTotal:  d.CurrentTotal.String(),
		TotalDelta:    d.TotalDelta.String(),
		Currency:      d.Currency.String(),
		Resources:     make([]deltaView, 0, len(d.Resources)),
	}
	for _, r := range d.Resources {
		view.Resources = append(view.Resources, deltaView{
			Kind:     r.Kind.String(),
			Resource: r.Resource.Label(),
			Baseline: r.Baseline.String(),
			Current:  r.Current.String(),
			Delta:    r.Delta.String(),
		})
	}
	b, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal diff: %w", err)
	}
	return string(b) + "\n", nil
}

// DecodeEstimate parses a JSON byte slice produced by [RenderJSON]
// back into a domain.Estimate. The on-disk contract is the view
// schema; we round-trip explicitly here so a baseline file saved by
// one c3x version still loads on the next.
//
// `c3x diff --baseline=path.json` lives on top of this — without a
// round-trippable JSON contract there's no diff command worth
// shipping.
func DecodeEstimate(raw []byte) (domain.Estimate, error) {
	var v estimateView
	if err := json.Unmarshal(raw, &v); err != nil {
		return domain.Estimate{}, fmt.Errorf("unmarshal estimate: %w", err)
	}
	cur, err := domain.ParseCurrency(v.Currency)
	if err != nil {
		return domain.Estimate{}, fmt.Errorf("baseline currency: %w", err)
	}
	costs, err := costsFromViews(v.Costs, cur)
	if err != nil {
		return domain.Estimate{}, err
	}
	total, err := decimal.NewFromString(v.ProjectTotal)
	if err != nil {
		return domain.Estimate{}, fmt.Errorf("project_total %q: %w", v.ProjectTotal, err)
	}
	ts, _ := time.Parse("2006-01-02T15:04:05Z", v.GeneratedAt)
	return domain.Estimate{
		Costs:        costs,
		ProjectTotal: total,
		Currency:     cur,
		GeneratedAt:  ts,
	}, nil
}

func costsFromViews(views []costView, cur domain.Currency) ([]domain.Cost, error) {
	out := make([]domain.Cost, 0, len(views))
	for _, v := range views {
		items := make([]domain.LineItem, 0, len(v.LineItems))
		for _, li := range v.LineItems {
			q, err := decimal.NewFromString(li.Quantity)
			if err != nil {
				return nil, fmt.Errorf("line item %q quantity: %w", li.Dimension, err)
			}
			r, err := decimal.NewFromString(li.UnitRate)
			if err != nil {
				return nil, fmt.Errorf("line item %q rate: %w", li.Dimension, err)
			}
			m, err := decimal.NewFromString(li.MonthlyCost)
			if err != nil {
				return nil, fmt.Errorf("line item %q cost: %w", li.Dimension, err)
			}
			items = append(items, domain.LineItem{
				Dimension:   li.Dimension,
				Description: li.Description,
				Unit:        li.Unit,
				Quantity:    q,
				UnitRate:    r,
				MonthlyCost: m,
				PriceSource: li.PriceSource,
			})
		}
		subtotal, err := decimal.NewFromString(v.MonthlySubtotal)
		if err != nil {
			return nil, fmt.Errorf("subtotal of %q: %w", v.Resource, err)
		}
		out = append(out, domain.Cost{
			Resource:        domain.Reference{Kind: v.Kind, Name: v.Name},
			LineItems:       items,
			MonthlySubtotal: subtotal,
			Currency:        cur,
		})
	}
	return out, nil
}

// View types render Decimals as strings so JSON consumers don't lose
// precision through a float round-trip. They mirror domain.* but are
// explicit because the on-disk contract should be stable independent
// of internal type churn.

type estimateView struct {
	Costs        []costView `json:"costs"`
	ProjectTotal string     `json:"project_total"`
	Currency     string     `json:"currency"`
	GeneratedAt  string     `json:"generated_at"`
	// CaveatCount is the number of caveats across the estimate: zero
	// means every line is a matched price for the resource as
	// configured. Always present so pipelines can gate on it.
	CaveatCount int `json:"caveat_count"`
}

type caveatView struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func toCaveatViews(cs []domain.Caveat) []caveatView {
	if len(cs) == 0 {
		return nil
	}
	out := make([]caveatView, len(cs))
	for i, c := range cs {
		out[i] = caveatView{Code: c.Code, Detail: c.Detail}
	}
	return out
}

type costView struct {
	Resource        string         `json:"resource"`
	Kind            string         `json:"kind"`
	Name            string         `json:"name"`
	LineItems       []lineItemView `json:"line_items"`
	MonthlySubtotal string         `json:"monthly_subtotal"`
	Currency        string         `json:"currency"`
	Caveats         []caveatView   `json:"caveats,omitempty"`
}

type lineItemView struct {
	Dimension   string       `json:"dimension"`
	Description string       `json:"description"`
	Unit        string       `json:"unit"`
	Quantity    string       `json:"quantity"`
	UnitRate    string       `json:"unit_rate"`
	MonthlyCost string       `json:"monthly_cost"`
	PriceSource string       `json:"price_source"`
	Caveats     []caveatView `json:"caveats,omitempty"`
}

type diffView struct {
	BaselineTotal string      `json:"baseline_total"`
	CurrentTotal  string      `json:"current_total"`
	TotalDelta    string      `json:"total_delta"`
	Currency      string      `json:"currency"`
	Resources     []deltaView `json:"resources"`
}

type deltaView struct {
	Kind     string `json:"kind"`
	Resource string `json:"resource"`
	Baseline string `json:"baseline"`
	Current  string `json:"current"`
	Delta    string `json:"delta"`
}

func toCostViews(costs []domain.Cost) []costView {
	out := make([]costView, 0, len(costs))
	for _, c := range costs {
		items := make([]lineItemView, 0, len(c.LineItems))
		for _, li := range c.LineItems {
			items = append(items, lineItemView{
				Dimension:   li.Dimension,
				Description: li.Description,
				Unit:        li.Unit,
				Quantity:    li.Quantity.String(),
				UnitRate:    li.UnitRate.String(),
				MonthlyCost: li.MonthlyCost.String(),
				PriceSource: li.PriceSource,
				Caveats:     toCaveatViews(li.Caveats),
			})
		}
		out = append(out, costView{
			Resource:        c.Resource.Label(),
			Kind:            c.Resource.Kind,
			Name:            c.Resource.Name,
			LineItems:       items,
			MonthlySubtotal: c.MonthlySubtotal.String(),
			Currency:        c.Currency.String(),
			Caveats:         toCaveatViews(c.ResourceCaveats),
		})
	}
	return out
}
