package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudwatchMetricAlarmRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_cloudwatch_metric_alarm",
		CoreRFunc: newCloudwatchMetricAlarm,
	}
}
func newCloudwatchMetricAlarm(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	comparisonOperator := d.Get("comparison_operator").String()

	var metricCount int64
	var period int64

	if len(d.Get("metric_query").Array()) > 0 {
		metricCount = 0
		for _, metric := range d.Get("metric_query.#.metric").Array() {
			metrics := metric.Array()

			if len(metrics) == 0 {
				continue
			}

			metricCount++

			for _, m := range metrics {
				if period == 0 && m.Get("period").Exists() {
					period = m.Get("period").Int()
				}
			}
		}
	} else {
		metricCount = 1
		period = d.Get("period").Int()
	}

	r := &aws.CloudwatchMetricAlarm{
		Address:            d.Address,
		Region:             region,
		ComparisonOperator: comparisonOperator,
		Metrics:            metricCount,
		Period:             period,
	}
	return r
}
