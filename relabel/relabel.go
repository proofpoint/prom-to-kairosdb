package relabel

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/prometheus/common/model"
	"github.com/proofpoint/prom-to-kairosdb/config"
)

//Process the samples and apply RelabelConfig
func Process(metric model.Metric, cfgs ...*config.RelabelConfig) model.Metric {
	for _, cfg := range cfgs {
		metric = relabel(metric, cfg)
		if metric == nil {
			return nil
		}
	}
	return metric
}

func relabel(metric model.Metric, cfg *config.RelabelConfig) model.Metric {
	values := make([]string, 0, len(cfg.SourceLabels))
	for _, labelName := range cfg.SourceLabels {
		values = append(values, string(metric[labelName]))
	}
	valueOfSourceLabels := strings.Join(values, cfg.Separator)

	switch cfg.Action {
	case config.RelabelDrop:
		if cfg.Regex.MatchString(valueOfSourceLabels) {
			logrus.Debug("dropping metric with values: ", valueOfSourceLabels)
			return nil
		}
	case config.RelabelKeep:
		if !cfg.Regex.MatchString(valueOfSourceLabels) {
			logrus.Debug("dropping metric with values: ", valueOfSourceLabels)
			return nil
		}
	case config.RelabelAddPrefix:
		if cfg.Regex.MatchString(valueOfSourceLabels) {
			metric[model.MetricNameLabel] = model.LabelValue(fmt.Sprintf("%s%s", cfg.Prefix, metric[model.MetricNameLabel]))
			logrus.Debugf("Added prefix [%s]: %s\n", cfg.Prefix, metric[model.MetricNameLabel])
		}
	case config.RelabelLabelDrop:
		for labelName := range metric {
			if cfg.Regex.MatchString(string(labelName)) {
				logrus.Debugf("dropping label [%s] in metric [%s]: ", labelName, string(metric["__name__"]))
				delete(metric, labelName)
			}
		}
	case config.RelabelLabelKeep:
		for labelName := range metric {
			if !cfg.Regex.MatchString(string(labelName)) {
				logrus.Debugf("dropping labels [%s] from metric [%s]", labelName, string(metric["__name__"]))
				delete(metric, labelName)
			}
		}
	default:
		logrus.Warnf("warn: retrieval.relabel: unknown relabel action type %s\n", cfg.Action)
	}
	return metric
}
