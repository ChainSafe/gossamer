package trie

//go:generate mockgen -destination=mock_metrics_test.go -package $GOPACKAGE . Metrics

// Metrics is the metrics interface to use for the trie(s).
type Metrics interface {
	NodesAdd(n uint32)
	NodesSub(n uint32)
}
