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
