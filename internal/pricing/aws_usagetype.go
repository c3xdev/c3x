package pricing

import "strings"

// awsUsagetypePrefix maps an AWS region to the prefix its usagetypes
// carry ("EU-AmazonEKS-Hours:perCluster" in eu-west-1). Taken from the
// pricing data itself, not from AWS documentation, which does not list
// them. us-east-1 is "USE1" for most services.
var awsUsagetypePrefix = map[string]string{
	"us-east-1":      "USE1",
	"us-east-2":      "USE2",
	"us-west-1":      "USW1",
	"us-west-2":      "USW2",
	"us-gov-east-1":  "UGE1",
	"us-gov-west-1":  "UGW1",
	"ca-central-1":   "CAN1",
	"ca-west-1":      "CAN2",
	"mx-central-1":   "MXC1",
	"sa-east-1":      "SAE1",
	"eu-west-1":      "EU",
	"eu-west-2":      "EUW2",
	"eu-west-3":      "EUW3",
	"eu-central-1":   "EUC1",
	"eu-central-2":   "EUC2",
	"eu-north-1":     "EUN1",
	"eu-south-1":     "EUS1",
	"eu-south-2":     "EUS2",
	"il-central-1":   "ILC1",
	"me-central-1":   "MEC1",
	"me-south-1":     "MES1",
	"af-south-1":     "AFS1",
	"ap-east-1":      "APE1",
	"ap-east-2":      "APE2",
	"ap-northeast-1": "APN1",
	"ap-northeast-2": "APN2",
	"ap-northeast-3": "APN3",
	"ap-south-1":     "APS3",
	"ap-south-2":     "APS5",
	"ap-southeast-1": "APS1",
	"ap-southeast-2": "APS2",
	"ap-southeast-3": "APS4",
	"ap-southeast-4": "APS6",
	"ap-southeast-5": "APS7",
	"ap-southeast-6": "APS8",
	"ap-southeast-7": "APS9",
	"cn-north-1":     "CNN1",
	"cn-northwest-1": "CNW1",
}

// awsKnownPrefixes is the set of region prefixes, to tell a prefixed
// usagetype ("EU-NatGateway-Hours") from an unprefixed one.
var awsKnownPrefixes = func() map[string]bool {
	m := make(map[string]bool, len(awsUsagetypePrefix))
	for _, p := range awsUsagetypePrefix {
		m[p] = true
	}
	return m
}()

// localizeAWSUsagetype rewrites a us-east-1 usagetype filter to the query
// region's equivalent. The catalog writes usagetypes as they appear in
// us-east-1, which is either with a USE1- prefix
// ("USE1-AmazonEKS-Hours:perCluster") or, for some services, with none
// ("Aurora:ServerlessV2Usage", "NatGateway-Hours"), while every other
// region prefixes them ("EU-Aurora:ServerlessV2Usage"). Without this,
// every other region missed and was priced at the us-east-1 rate. ok is
// false when there is nothing to rewrite. A rewrite can be wrong for a
// service whose usagetypes are unprefixed everywhere; the caller retries
// the original query when the rewritten one misses.
func localizeAWSUsagetype(q Query) (Query, bool) {
	if q.Provider != "aws" || q.Region == "us-east-1" {
		return q, false
	}
	prefix, known := awsUsagetypePrefix[q.Region]
	if !known {
		return q, false
	}
	var out []KV
	for i, f := range q.AttributeFilters {
		if !strings.EqualFold(f.Key, "usagetype") || f.Value == "" {
			continue
		}
		var localized string
		if rest, ok := strings.CutPrefix(f.Value, "USE1-"); ok {
			localized = prefix + "-" + rest
		} else if head, _, _ := strings.Cut(f.Value, "-"); !awsKnownPrefixes[head] {
			localized = prefix + "-" + f.Value
		} else {
			continue // already names a region
		}
		if out == nil {
			out = append([]KV(nil), q.AttributeFilters...)
		}
		out[i].Value = localized
	}
	if out == nil {
		return q, false
	}
	q.AttributeFilters = out
	return q, true
}
