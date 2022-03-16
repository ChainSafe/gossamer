package metrics

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var _ Metrics = (*Prometheus)(nil)

type Prometheus struct {
	nodesGauge prometheus.Gauge
}

func NewPrometheus() (metrics *Prometheus, err error) {
	metrics = new(Prometheus)
	err = metrics.setupDefaults()
	if err != nil {
		return metrics, err
	}

	return metrics, nil
}

func (m *Prometheus) setupDefaults() (err error) {
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

func (m *Prometheus) NodesAdd(n int) {
	m.nodesGauge.Add(float64(n))
}
