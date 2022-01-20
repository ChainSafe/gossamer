// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package metrics

import (
	"net/http"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const defaultInterval = 10 * time.Second

var logger log.LeveledLogger = log.NewFromGlobal(log.AddContext("pkg", "metrics"))

// Config is base config for metrics
type Config struct {
	Publish bool
}

// IntervalConfig for interval collection
type IntervalConfig struct {
	Config
	Interval time.Duration
}

// NewIntervalConfig is constructor for IntervalConfig, and uses default metrics interval
func NewIntervalConfig(publish bool) IntervalConfig {
	return IntervalConfig{
		Config:   Config{publish},
		Interval: defaultInterval,
	}
}

// Start will start a dedicated metrics server at the given address.
func Start(address string) {
	logger.Info("Starting metrics server at http://" + address + "/metrics")
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(address, nil); err != nil {
			log.Errorf("Metrics HTTP server crashed: %s", err)
		}
	}()
}
