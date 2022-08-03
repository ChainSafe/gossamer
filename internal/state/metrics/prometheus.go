package metrics

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus struct {
	nodeAdditions prometheus.Counter
	nodeDeletions prometheus.Counter
}

func NewPrometheus() (metrics *Prometheus, err error) {
	metrics = new(Prometheus)
	collectorsToRegister := make(map[string]prometheus.Collector)

	metrics.nodeAdditions = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "gossamer_storage",
		Name:      "state_tries_node_additions",
		Help:      "creation of nodes in all the state tries in memory",
	})
	collectorsToRegister["node additions counter"] = metrics.nodeAdditions

	metrics.nodeDeletions = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "gossamer_storage",
		Name:      "state_tries_node_deletions",
		Help:      "deletion of nodes in all the state tries in memory",
	})
	collectorsToRegister["node deletions counter"] = metrics.nodeDeletions

	for collectorName, collectorToRegister := range collectorsToRegister {
		err = prometheus.Register(collectorToRegister)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return nil, fmt.Errorf("cannot register %s: %w", collectorName, err)
		}
	}

	return metrics, nil
}

func (m *Prometheus) NodesAdded(n uint32) {
	m.nodeAdditions.Add(float64(n))
}

func (m *Prometheus) NodesDeleted(n uint32) {
	m.nodeAdditions.Add(-float64(n))
}
