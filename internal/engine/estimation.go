package engine

import "context"

// UsageEstimator queries cloud providers to gather past usage information, then
// projects future usage based on the results.
type UsageEstimator func(context.Context, map[string]interface{}) error

// Remediater allows correction of cloud configuration issues
// so that future runs of C3X will provide more accurate results.
type Remediater interface {
	// Describe provides an English description of the remediation action X that
	// would fit into a sentence "May we X?" (e.g. "enable bucket metrics").
	// The description can be used to prompt the user before taking action.
	Describe() string

	// Remediate attempts to fix a problem in the cloud that prevents estimation,
	// e.g. by enabling metrics collection on certain resources.
	Remediate() error
}
