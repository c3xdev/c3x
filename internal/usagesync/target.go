package usagesync

import (
	"sort"

	"github.com/c3xdev/c3x/internal/domain"
)

// Resolve decides, for every resource of a kind the source supports, the
// cloud identifier and region to look it up by.
//
// resources are what the estimate sees, so their addresses are the
// resource_usage keys. state, when given, is the Terraform state and is
// the source of truth for the identifier and region: it holds what was
// actually created. Without it, a literal in configuration is used. A
// resource whose identifier or region cannot be established is left out
// and explained in the returned map, never guessed.
func Resolve(resources, state []domain.Resource, haveState bool, fallbackRegion string, emits map[string][]string) ([]Target, map[string]string) {
	byAddress := make(map[string]domain.Resource, len(state))
	for _, s := range state {
		byAddress[s.Ref.Label()] = s
	}

	var targets []Target
	problems := map[string]string{}
	for _, r := range resources {
		if _, ok := emits[r.Ref.Kind]; !ok {
			continue
		}
		addr := r.Ref.Label()
		id, region, problem := identify(r, byAddress, haveState, fallbackRegion)
		if problem != "" {
			problems[addr] = problem
			continue
		}
		targets = append(targets, Target{Address: addr, Kind: r.Ref.Kind, ID: id, Region: region})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Address < targets[j].Address })
	return targets, problems
}

// idAttribute is the attribute that names a resource in the cloud.
var idAttribute = map[string]string{"aws_s3_bucket": "bucket"}

func identify(r domain.Resource, state map[string]domain.Resource, haveState bool, fallbackRegion string) (id, region, problem string) {
	attr := idAttribute[r.Ref.Kind]
	src := r
	if haveState {
		s, ok := state[r.Ref.Label()]
		if !ok {
			return "", "", "not in the Terraform state; apply it first, or leave out --state to use the name in configuration"
		}
		src = s
	}
	id, _ = src.Attributes[attr].(string)
	if id == "" {
		if haveState {
			return "", "", "the Terraform state has no " + attr + " for it"
		}
		return "", "", attr + " is not a literal in configuration; pass --state"
	}

	switch {
	case src.Region != nil && *src.Region != "":
		region = *src.Region
	case r.Region != nil && *r.Region != "":
		region = *r.Region
	default:
		region = fallbackRegion
	}
	if region == "" {
		return "", "", "region unknown; pass --region"
	}
	return id, region, ""
}
