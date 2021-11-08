// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package metrics

import (
	"net/http"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	ethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/prometheus"
)

var logger log.LeveledLogger = log.NewFromGlobal(log.AddContext("pkg", "metrics"))

const (
	// RefreshInterval is the refresh time for publishing metrics.
	RefreshInterval = time.Second * 10
	refreshFreq     = int64(RefreshInterval / time.Second)
)

// PublishMetrics function will export the /metrics endpoint to prometheus process
func PublishMetrics(address string) {
	ethmetrics.Enabled = true
	setupMetricsServer(address)
}

// setupMetricsServer starts a dedicated metrics server at the given address.
func setupMetricsServer(address string) {
	m := http.NewServeMux()
	m.Handle("/metrics", prometheus.Handler(ethmetrics.DefaultRegistry))
	logger.Info("Starting metrics server at http://" + address + "/metrics")
	go func() {
		if err := http.ListenAndServe(address, m); err != nil {
			logger.Errorf("Metrics HTTP server crashed: %s", err)
		}
	}()
}
