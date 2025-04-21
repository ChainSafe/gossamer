package bitfieldsigning

//go:generate mockgen -destination=mocks_blockstate_test.go -package=$GOPACKAGE . BlockState
//go:generate mockgen -destination=mocks_instance_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
