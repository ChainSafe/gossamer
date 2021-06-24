package metrics

import (
	"time"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

type GaugeCollector interface {
	Update() int64
}

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
