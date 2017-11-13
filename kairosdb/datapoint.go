package kairosdb

import (
	"math"

	"github.com/prometheus/common/model"
	"github.com/proofpoint/prom-to-kairosdb/config"
	"github.com/proofpoint/prom-to-kairosdb/relabel"
)

// DataPoint represents the kairosdb DataPoint
type DataPoint struct {
	Name      string
	Timestamp int64
	Value     float64
	Tags      map[string]string
}

// ValidValue filters out values which are not supported by KairosDB
func ValidValue(value float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}

	return true
}

func FilterAndProcessSamples(samples model.Samples, cfg *config.Config) (datapoints []*DataPoint) {
	for _, sample := range samples {
		metric := sample.Metric
		value := float64(sample.Value)
		timestamp := int64(sample.Timestamp)

		metric = relabel.Process(metric, cfg.MetricRelabelConfigs...)
		if metric == nil {
			continue
		}

		if !ValidValue(value) {
			continue
		}

		tags := tagsFromMetric(metric)
		datapoints = append(datapoints, &DataPoint{
			Name:      string(metric[model.MetricNameLabel]),
			Timestamp: timestamp,
			Value:     value,
			Tags:      tags,
		})
	}
	return
}

func tagsFromMetric(metric model.Metric) map[string]string {
	tags := make(map[string]string, len(metric)-1)
	for labelName, labelValue := range metric {
		if labelName == model.MetricNameLabel {
			continue
		}

		if string(labelValue) == "" {
			continue
		}

		tags[string(labelName)] = string(labelValue)
	}
	return tags
}
