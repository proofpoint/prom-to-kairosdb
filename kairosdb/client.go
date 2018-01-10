package kairosdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/proofpoint/prom-to-kairosdb/config"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
)

var (
	sentSamples = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sent_samples_total",
			Help: "Total number of processed samples sent to remote storage.",
		},
		[]string{"remote"},
	)
	failedSamples = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "failed_samples_total",
			Help: "Total number of processed samples which failed on send to remote storage.",
		},
		[]string{"remote"},
	)
	unknownStatusSamples = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "unknown_status_samples_total",
			Help: "Total number of samples sent without receiving a response.",
		},
		[]string{"remote"},
	)
	sentBatchDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sent_batch_duration_seconds",
			Help:    "Duration of sample batch send calls to the remote storage.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"remote"},
	)
	filteredSamples = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "filtered_samples_total",
			Help: "Total number of samples which got filtered out before being sent to remote storage.",
		},
		[]string{"remote"},
	)
)

func RegisterPrometheusMetrics() {
	prometheus.MustRegister(sentSamples)
	prometheus.MustRegister(failedSamples)
	prometheus.MustRegister(unknownStatusSamples)
	prometheus.MustRegister(sentBatchDuration)
	prometheus.MustRegister(filteredSamples)
}

const (
	postEndpoint    = "/api/v1/datapoints"
	contentTypeJSON = "application/json"
)

// Client struct defined how to connect to kairosdb
type Client struct {
	cfg     *config.Config
	url     config.URL
	timeout time.Duration
}

// NewClient returns a new client for KairosDB
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg:     cfg,
		url:     cfg.KairosdbURL,
		timeout: cfg.Timeout,
	}
}

// Send - Apply RelabelConfigs, massage the data and write the samples to KairosDB
func (c *Client) Send(samples model.Samples) (err error) {
	datapoints := FilterAndProcessSamples(samples, c.cfg)

	filteredSamplesCount := len(samples) - len(datapoints)
	filteredSamples.WithLabelValues(c.name()).Add(float64(filteredSamplesCount))

	begin := time.Now()

	err = c.write(datapoints)
	if err != nil {
		logrus.Errorf("failed writing metrics to downstream. error: %s", err)
	}

	duration := time.Since(begin).Seconds()
	sentBatchDuration.WithLabelValues(c.name()).Observe(duration)

	return
}

// Write sends a batch of datapoints to KairosDB via its HTTP API.
func (c *Client) write(datapoints []*DataPoint) error {
	totalRequests := len(datapoints)

	c.url.Path = postEndpoint
	buf, err := json.Marshal(datapoints)
	if err != nil {
		return err
	}

	logrus.Debugf("pushing %d datapoints", totalRequests)
	if c.cfg.DryRun {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	resp, err := ctxhttp.Post(ctx, http.DefaultClient, c.url.String(), contentTypeJSON, bytes.NewBuffer(buf))

	if err != nil {
		failedSamples.WithLabelValues(c.name()).Add(float64(totalRequests))
		return err
	}

	defer resp.Body.Close()

	if resp == nil {
		failedSamples.WithLabelValues(c.name()).Add(float64(totalRequests))
		return fmt.Errorf("no response received")
	}

	if resp.StatusCode == http.StatusNoContent {
		logrus.Infof("pushed %d datapoints successfully", totalRequests)
		sentSamples.WithLabelValues(c.name()).Add(float64(totalRequests))
		return nil
	}

	// API returns status code 400 on error, encoding error details in the
	// response content in JSON.
	respbuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("%s", err)
		unknownStatusSamples.WithLabelValues(c.name()).Add(float64(totalRequests))
		return err
	}

	var r map[string][]interface{}
	if err = json.Unmarshal(respbuf, &r); err != nil {
		logrus.Errorf("response received is : %s", string(respbuf))
		logrus.Errorf("%s", err)
		unknownStatusSamples.WithLabelValues(c.name()).Add(float64(totalRequests))
		return err
	}

	failed := len(r["errors"])
	successful := totalRequests - failed

	//unlikely, but will keep it here anyways
	//added because a code issue was causing this condition
	if successful < 0 {
		logrus.Errorf("response from kairosdb %v", r)
		logrus.Errorf("req to kairosdb %v", string(buf))
		return fmt.Errorf("number of failed datapoints [%d] is greater than total datapoints [%d]", failed, totalRequests)
	}

	sentSamples.WithLabelValues(c.name()).Add(float64(successful))
	failedSamples.WithLabelValues(c.name()).Add(float64(failed))

	return fmt.Errorf("failed to write [%d] samples of [%d]", failed, totalRequests)
}

func (c *Client) name() string {
	return "kairosdb"
}
