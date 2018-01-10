package kairosdb

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/proofpoint/prom-to-kairosdb/config"
	"github.com/stretchr/testify/assert"
)

func TestValidValue(t *testing.T) {
	cases := []struct {
		name     string
		value    float64
		expected bool
	}{
		{
			name:     "NaN value",
			value:    math.NaN(),
			expected: false,
		},
		{
			name:     "infinite value",
			value:    math.Inf(0),
			expected: false,
		},
		{
			name:     "random valid value",
			value:    rand.Float64(),
			expected: true,
		},
	}

	for _, c := range cases {
		actual := ValidValue(c.value)
		assert.Equal(t, c.expected, actual)
	}
}

func TestFilterAndProcessSamples(t *testing.T) {
	randvalue1 := rand.Float64()
	randvalue2 := rand.Float64()

	timevalue1 := time.Now().UnixNano()
	timevalue2 := time.Now().UnixNano()

	samples := []*model.Sample{
		{
			Metric: model.Metric{
				model.LabelName("label1"):   model.LabelValue("value1"),
				model.LabelName("__name__"): model.LabelValue("metricname-1"),
				model.LabelName("label2"):   model.LabelValue("value2"),
			},
			Value:     model.SampleValue(randvalue1),
			Timestamp: model.Time(timevalue1),
		},
		{
			Metric: model.Metric{
				model.LabelName("label3"):   model.LabelValue("value3"),
				model.LabelName("__name__"): model.LabelValue("metricname-2"),
				model.LabelName("label4"):   model.LabelValue("value4"),
			},
			Value:     model.SampleValue(randvalue2),
			Timestamp: model.Time(timevalue2),
		},
		{
			Metric: model.Metric{
				model.LabelName("label3"):   model.LabelValue("value3"),
				model.LabelName("__name__"): model.LabelValue("metricname-2"),
				model.LabelName("label4"):   model.LabelValue("value4"),
			},
			Value:     model.SampleValue(math.Inf(0)),
			Timestamp: model.Time(timevalue2),
		},
		{
			Metric: model.Metric{
				model.LabelName("label3"):   model.LabelValue("value3"),
				model.LabelName("__name__"): model.LabelValue("metricname-2"),
				model.LabelName("label4"):   model.LabelValue("value4"),
			},
			Value:     model.SampleValue(math.NaN()),
			Timestamp: model.Time(timevalue2),
		},
	}

	datapoints := []*DataPoint{
		{
			Name:      "my-prefix.metricname-1",
			Value:     randvalue1,
			Timestamp: timevalue1,
			Tags: map[string]string{
				"label1": "value1",
				"label2": "value2",
			},
		},
		{
			Name:      "my-prefix.metricname-2",
			Value:     randvalue2,
			Timestamp: timevalue2,
			Tags: map[string]string{
				"label3": "value3",
				"label4": "value4",
			},
		},
	}

	cases := []struct {
		name       string
		samples    model.Samples
		cfgfile    string
		datapoints []*DataPoint
	}{
		{
			name:       "config with toplevel prefix",
			samples:    samples,
			datapoints: datapoints,
			cfgfile:    "testdata/config.yaml",
		},
	}

	for _, c := range cases {
		cfg, err := config.ParseCfgFile(c.cfgfile)

		if err != nil {
			t.Errorf("case %s. failed to parse config file %s", c.name, c.cfgfile)
		}

		actual := FilterAndProcessSamples(c.samples, cfg)
		assert.Equal(t, c.datapoints, actual)
	}
}
