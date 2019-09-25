package transaction

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
)

type Pool map[common.Hash]*ValidTransaction

type Queue interface {
	Pop() *ValidTransaction
	Insert(vt *ValidTransaction)
}

type TransactionTag []byte

// see: https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/sr-primitives/src/transaction_validity.rs#L178
type Validity struct {
	priority uint64
	requires  []TransactionTag
	provides  []TransactionTag
	longevity uint64
	propagate bool
}

func NewValidity(priority uint64, requires, provides []TransactionTag, longevity uint64, propagate bool) *Validity {
	return &Validity{
		priority: priority,
		requires: requires,
		provides: provides,
		longevity: longevity,
		propagate: propagate,
	}
}

type ValidTransaction struct {
	extrinsic types.Extrinsic
	validity  *Validity
}

func NewValidTransaction(extrinsic types.Extrinsic, validity *Validity) *ValidTransaction {
	return &ValidTransaction{
		extrinsic: extrinsic,
		validity:  validity,
	}
}
