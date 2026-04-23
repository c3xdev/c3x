package engine

import (
	"time"
)

type ObservedCost struct {
	ResourceID     string
	StartTimestamp time.Time
	EndTimestamp   time.Time
	CostComponents []*LineItem
}
