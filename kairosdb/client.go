package kairosdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/proofpoint/prom-to-kairosdb/config"
	"golang.org/x/net/context/ctxhttp"
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
			Help: "Total number of samples which got filtered before being sent to remote storage.",
		},
		[]string{"remote"},
	)
)

func RegisterPrometheusMetrics() {
	prometheus.MustRegister(sentSamples)
	prometheus.MustRegister(failedSamples)
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

// SendSamples - sends the samples to KairosDB
func (c *Client) Send(samples model.Samples) (err error) {
	datapoints := FilterAndProcessSamples(samples, c.cfg)

	filteredSamplesCount := len(samples) - len(datapoints)
	filteredSamples.WithLabelValues(c.name()).Add(float64(filteredSamplesCount))

	begin := time.Now()

	err = c.write(datapoints)

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

	if c.cfg.DryRun {
		fmt.Printf("pushing %d datapoints : %v", totalRequests, string(buf))
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	resp, err := ctxhttp.Post(ctx, http.DefaultClient, c.url.String(), contentTypeJSON, bytes.NewBuffer(buf))

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	// API returns status code 400 on error, encoding error details in the
	// response content in JSON.
	buf, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var r map[string][]interface{}
	if err := json.Unmarshal(buf, &r); err != nil {
		return err
	}

	failed := len(r["errors"])
	successful := totalRequests - failed

	sentSamples.WithLabelValues(c.name()).Add(float64(successful))
	failedSamples.WithLabelValues(c.name()).Add(float64(failed))

	return fmt.Errorf("failed to write %d samples to KairosDB, %d succeeded", len(r["errors"]), totalRequests-len(r["errors"]))
}

func (c *Client) name() string {
	return "kairosdb"
}
