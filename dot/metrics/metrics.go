// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package metrics

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/ChainSafe/log15"
	ethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/prometheus"
)

var logger = log.New("pkg", "metrics")

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
	logger.Info("Starting metrics server", "addr", fmt.Sprintf("http://%s/metrics", address))
	go func() {
		if err := http.ListenAndServe(address, m); err != nil {
			log.Error("Failure in running metrics server", "err", err)
		}
	}()
}
