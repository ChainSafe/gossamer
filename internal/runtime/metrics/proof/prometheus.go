package proof

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
		Namespace: "gossamer_runtime",
		Name:      "temp_proof_tries_node_additions",
		Help:      "additions of nodes in all the temporary tries in memory used for proofs",
	})
	collectorsToRegister["temporary proof tries node additions counter"] = metrics.nodeAdditions

	metrics.nodeDeletions = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "gossamer_runtime",
		Name:      "temp_proof_tries_node_deletions",
		Help:      "deletions of nodes in all the temporary tries in memory used for proofs",
	})
	collectorsToRegister["temporary proof tries node deletions counter"] = metrics.nodeDeletions

	for collectorName, collectorToRegister := range collectorsToRegister {
		err = prometheus.Register(collectorToRegister)
		if err != nil && !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return nil, fmt.Errorf("registering %s: %w", collectorName, err)
		}
	}

	return metrics, nil
}

func (p *Prometheus) NodesAdded(n uint32) {
	p.nodeAdditions.Add(float64(n))
}

func (p *Prometheus) NodesDeleted(n uint32) {
	p.nodeDeletions.Add(float64(n))
}
