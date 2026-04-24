package localpricing

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
)

// QueryResult represents a single product price result from the local database.
type QueryResult struct {
	SKU        string
	Attributes map[string]string
	Prices     []PriceEntry
}

// PriceEntry represents a single price dimension.
type PriceEntry struct {
	USD         string `json:"USD"`
	Unit        string `json:"unit"`
	Description string `json:"description,omitempty"`
}

// LookupPrice queries the local pricing database for a product matching the given filters.
// Returns a gjson.Result in the same format as the remote GraphQL API response.
func (s *Store) LookupPrice(vendor, service, region string, attrFilters map[string]string, priceUnit string) (gjson.Result, error) {
	query := `SELECT attributes, prices FROM products WHERE vendor = ? AND service = ?`
	args := []interface{}{vendor, service}

	if region != "" {
		query += ` AND region = ?`
		args = append(args, region)
	}

	// Push attribute filters to SQL via json_extract for efficient filtering.
	// The in-memory matchesAttributes pass in findMatchingProduct stays as a
	// correctness safety net for edge cases.
	for key, value := range attrFilters {
		query += ` AND json_extract(attributes, ?) = ?`
		args = append(args, "$."+key, value)
	}

	query += ` LIMIT 10`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return gjson.Result{}, err
	}
	defer rows.Close()

	return findMatchingProduct(rows, attrFilters, priceUnit)
}

func findMatchingProduct(rows *sql.Rows, attrFilters map[string]string, priceUnit string) (gjson.Result, error) {
	for rows.Next() {
		var attrsJSON, pricesJSON string
		if err := rows.Scan(&attrsJSON, &pricesJSON); err != nil {
			continue
		}

		attrs := gjson.Parse(attrsJSON)
		if matchesAttributes(attrs, attrFilters) {
			// Build a result that looks like the GraphQL API response
			var prices []PriceEntry
			if err := json.Unmarshal([]byte(pricesJSON), &prices); err != nil {
				continue
			}

			// Filter by priceUnit if specified
			if priceUnit != "" {
				filtered := prices[:0]
				for _, p := range prices {
					if p.Unit == priceUnit {
						filtered = append(filtered, p)
					}
				}
				prices = filtered
			}

			if len(prices) > 0 {
				// Use json.Marshal instead of fmt.Sprintf to safely handle
				// special characters in price fields (quotes, backslashes).
				out, _ := json.Marshal(map[string]interface{}{
					"data": map[string]interface{}{
						"products": []interface{}{
							map[string]interface{}{
								"prices": []interface{}{
									map[string]string{"USD": prices[0].USD, "unit": prices[0].Unit},
								},
							},
						},
					},
				})
				return gjson.ParseBytes(out), nil
			}
		}
	}

	return gjson.Result{}, nil
}

func matchesAttributes(attrs gjson.Result, filters map[string]string) bool {
	for key, value := range filters {
		actual := attrs.Get(key).String()
		if actual == "" {
			return false
		}
		if !strings.EqualFold(actual, value) {
			return false
		}
	}
	return true
}
