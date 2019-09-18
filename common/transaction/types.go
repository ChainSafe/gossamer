package transaction

import (
	"github.com/ChainSafe/gossamer/common"
)

type Pool map[common.Hash]common.Extrinsic

type Queue interface {
	Pop() *ValidTransaction
	Insert(vt *ValidTransaction)
}

// see: https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/sr-primitives/src/transaction_validity.rs#L178
type Validity struct {
	priority  uint64
	requires  []byte
	provides  []byte
	longevity uint64
	propagate bool
}

type ValidTransaction struct {
	extrinsic common.Extrinsic
	validity  Validity
}
