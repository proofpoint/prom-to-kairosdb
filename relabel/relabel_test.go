package relabel

import (
	"reflect"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/rajatjindal/prom-to-kairosdb/config"
)

func TestRelabel(t *testing.T) {
	cases := []struct {
		name    string
		input   model.Metric
		relabel []*config.RelabelConfig
		output  model.Metric
		err     error
	}{
		{
			name: "no relabel config",
			input: model.Metric{
				"a1":       "v1",
				"a2":       "v2",
				"__name__": "metricname",
			},
			output: model.Metric{
				"a1":       "v1",
				"a2":       "v2",
				"__name__": "metricname",
			},
		},
		{
			name: "prefix is added successfully",
			input: model.Metric{
				"a1":       "v1",
				"a2":       "v2",
				"__name__": "metricname",
			},
			relabel: []*config.RelabelConfig{
				{
					SourceLabels: model.LabelNames{"a1"},
					Regex:        config.MustNewRegexp("v1"),
					Action:       "addprefix",
					Prefix:       "validprefix.",
				},
			},
			output: model.Metric{
				"a1":       "v1",
				"a2":       "v2",
				"__name__": "validprefix.metricname",
			},
		},
		{
			name: "valid regex match, action drop",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					SourceLabels: model.LabelNames{"a1"},
					Regex:        config.MustNewRegexp("v1"),
					Action:       "drop",
				},
			},
			output: nil,
		},
		{
			name: "valid regex match, action keep",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					SourceLabels: model.LabelNames{"a1"},
					Regex:        config.MustNewRegexp("v1"),
					Action:       "keep",
				},
			},
			output: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
		},
		{
			name: "valid regex mismatch, action drop",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					SourceLabels: model.LabelNames{"a1"},
					Regex:        config.MustNewRegexp("v2"),
					Action:       "drop",
				},
			},
			output: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
		},
		{
			name: "valid regex match, multiple sourcelabels, action keep",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					SourceLabels: model.LabelNames{"a1", "a2"},
					Regex:        config.MustNewRegexp("v1v2"),
					Action:       "keep",
				},
			},
			output: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
		},
		{
			name: "valid regex match, multiple sourcelabels, action drop",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					SourceLabels: model.LabelNames{"a1", "a2"},
					Regex:        config.MustNewRegexp("v2v1"),
					Action:       "keep",
				},
			},
			output: nil,
		},
		{
			name: "valid regex match, multiple sourcelabels with separator, action drop",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					SourceLabels: model.LabelNames{"a1", "a2"},
					Separator:    ";",
					Regex:        config.MustNewRegexp("v1;v2"),
					Action:       "drop",
				},
			},
			output: nil,
		},
		{
			name: "valid regex mismatch, multiple sourcelabels with wrong separator, action drop",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					SourceLabels: model.LabelNames{"a1", "a2"},
					Separator:    "=",
					Regex:        config.MustNewRegexp("v1;v2"),
					Action:       "drop",
				},
			},
			output: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
		},
		{
			name: "valid regex match, multiple sourcelabels, action labeldrop",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					Regex:  config.MustNewRegexp("a1"),
					Action: "labeldrop",
				},
			},
			output: model.Metric{
				"a2": "v2",
			},
		},
		{
			name: "valid regex match, multiple sourcelabels, action labelkeep",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					Regex:  config.MustNewRegexp("a1"),
					Action: "labelkeep",
				},
			},
			output: model.Metric{
				"a1": "v1",
			},
		},
		{
			name: "valid regex match, unknown action",
			input: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
			relabel: []*config.RelabelConfig{
				{
					Regex:  config.MustNewRegexp("a1"),
					Action: "unknownaction",
				},
			},
			output: model.Metric{
				"a1": "v1",
				"a2": "v2",
			},
		},
	}

	for _, c := range cases {
		ao := Process(c.input, c.relabel...)

		if !reflect.DeepEqual(c.output, ao) {
			t.Errorf("case '%s'. Expected %+v, got %+v", c.name, c.output, ao)
		}
	}
}
