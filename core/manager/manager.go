package transaction

import (
	log "github.com/ChainSafe/log15"

	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/common"
	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/consensus/babe"
	"github.com/ChainSafe/gossamer/runtime"
)

type Manager struct {
	rt *runtime.Runtime
	b  *babe.Session
}

// ProcessTransaction attempts to validates the transaction
// if it is validated, it is added to the transaction pool of the BABE session
func (m *Manager) ProcessTransaction(e common.Extrinsic) {
	validity, err := m.validateTransaction(e)
	if err != nil {
		log.Error("ProcessTransaction", "error", err)
		return
	}

	vtx := &tx.NewValidTransaction(e, validity)
	b.TxQueue.Insert(vtx)

	return
}

// ProcessBlock attempts to add a block to the chain by calling `core_execute_block`
// if the block is validated, it is stored in the block DB and becomes part of the canonical chain
func (m *Manager) ProcessBlock(b *common.BlockHeader) {
	return
}

// runs the extrinsic through runtime function TaggedTransactionQueue_validate_transaction
// and returns *Validity
func (m *Manager) validateTransaction(e common.Extrinsic) (*tx.Validity, error) {
	ret, err := r.Exec("TaggedTransactionQueue_validate_transaction", 1, 0)
	if err != nil {
		return nil, err
	}

	v := new(tx.Validity)
	_, err = scale.Decode(ret, v)
	return v, err
}
