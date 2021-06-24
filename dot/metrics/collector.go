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
	"time"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

// GaugeCollector abstracts the Update function to collectors
// just implement the collection
type GaugeCollector interface {
	Update() int64
}

// CollectGaugeMetrics receives an timeout, label and a gauge collector
// and acquire the metrics timeout by timeout to a ethereum metrics gauge
func CollectGaugeMetrics(timeout time.Duration, label string, c GaugeCollector) {
	t := time.NewTicker(timeout)
	defer t.Stop()

	collectGauge(label, c)

	for range t.C {
		collectGauge(label, c)
	}
}

func collectGauge(label string, c GaugeCollector) {
	ethmetrics.Enabled = true
	pooltx := ethmetrics.GetOrRegisterGauge(label, nil)
	pooltx.Update(c.Update())
}
