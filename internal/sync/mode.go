package sync

type mode byte

const (
	bootstrap mode = iota
	tip
)

func (s mode) String() string {
	switch s {
	case bootstrap:
		return "bootstrap"
	case tip:
		return "tip"
	default:
		return "unknown"
	}
}
