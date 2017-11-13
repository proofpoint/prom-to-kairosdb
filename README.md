# prom-to-kairosdb

 [![Build Status](https://travis-ci.org/proofpoint/prom-to-kairosdb.svg?branch=master)](https://travis-ci.org/proofpoint/prom-to-kairosdb) [![Go Report Card](https://goreportcard.com/badge/github.com/proofpoint/prom-to-kairosdb)](https://goreportcard.com/report/github.com/proofpoint/prom-to-kairosdb)

 prom-to-kairosdb is a remote storage adapter for Prometheus that listens for metrics from Prometheus remote write service, and pushes to downstream KairosDB.

 It exposes following end points:

| Endpoint | Details |
| ------ | ------ |
| `/write` | Listens to metrics from Prometheus, reformat them, and push to KairosDB |
| `/metrics` | exposed the metrics for the prom-to-kairosdb itself |

By default the service starts on port `9201`.

# Relabeling
Like Prometheus, this service also supports a few relabeling features. e.g. if you want to drop an unwanted metric or keep only specific metrics or rename the metric itself etc.

Following are the Action it supports:

| Action | Details | Example |
| ------ | ------ | ------|
| `keep` | drops any metrics for which the provided sourcelabels `does not` matches the regex. ||
| `drop` | drops any metrics for which the provided sourcelabels matches the regex. ||
| `labelkeep` | drops any label not matching the regex. ||
| `labeldrop` | drops any label matching the regex. ||
| `addprefix` | Adds prefix to the metric name that matches the regex. ||

# Examples
#### drop the metrics that matches regex
```yaml
#drop the metric if metricname (identified by __name__) matches regex 'my_too_large_metric'
metric_relabel_configs:
   - source_labels: [ __name__ ]
     regex: 'my_too_large_metric'
     action: drop
```
#### keep the metric that matches regex (drop everything else)
```yaml
#keep the metric if metricname (identified by __name__) matches regex 'my_imp_metric'
metric_relabel_configs:
   - source_labels: [ __name__ ]
     regex: 'my_imp_metric'
     action: keep
```
#### drop the labels that matches regex
```yaml
#drop the label that does match regex 'label-not-needed'
metric_relabel_configs:
   - regex: 'label-not-needed'
     action: labeldrop
```
#### keep the labels that match regex
```yaml
#drop the labels that does not match regex 'label-not-needed'
metric_relabel_configs:
   - regex: 'label-needed'
     action: labelkeep
```
#### Add prefix to the metricname where sourcelabels values match regex
```yaml
#Add the prefix to the metric name if the value of tagName in the metric tags, matches the regex 'tagValue'
metric_relabel_configs:
   - source_labels: [ tagName ]
     regex: 'tagValue'
     action: keep
```
