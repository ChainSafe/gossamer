package prometheus

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	nodesGauge prometheus.Gauge
}

func New() (metrics *Metrics, err error) {
	metrics = new(Metrics)
	err = metrics.setupDefaults()
	if err != nil {
		return metrics, err
	}

	return metrics, nil
}

func (m *Metrics) setupDefaults() (err error) {
	collectorsToRegister := map[string]prometheus.Collector{}
	if m.nodesGauge == nil {
		m.nodesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "gossamer_storage",
			Name:      "nodes_cached_total",
			Help:      "total number of nodes in all the tries in memory",
		})
		collectorsToRegister["nodes gauge"] = m.nodesGauge
	}

	for collectorName, collectorToRegister := range collectorsToRegister {
		err = prometheus.Register(collectorToRegister)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return fmt.Errorf("cannot register %s gauge: %w", collectorName, err)
		}
	}

	return nil
}

func (m *Metrics) NodesAdd(n uint32) {
	m.nodesGauge.Add(float64(n))
}

func (m *Metrics) NodesSub(n uint32) {
	m.nodesGauge.Sub(float64(n))
}
