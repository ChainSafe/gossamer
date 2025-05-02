package util

//go:generate mockgen -destination=mock_blockstate_test.go -package $GOPACKAGE . BlockState
//go:generate mockgen -destination=mock_runtime_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
