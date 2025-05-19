package network

type ValidationVersion byte

const (
	ValidationVersionV1 ValidationVersion = iota + 1
	ValidationVersionV2
	ValidationVersionV3
)
