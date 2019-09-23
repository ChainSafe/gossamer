package manager

import (
	log "github.com/ChainSafe/log15"

	scale "github.com/ChainSafe/gossamer/codec"
	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/consensus/babe"
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/runtime"
)

type Service struct {
	rt *runtime.Runtime
	b  *babe.Session

	msgChan <-chan p2p.Message
}

func NewService(rt *runtime.Runtime, b *babe.Session, msgChan <-chan p2p.Message) *Service {
	return &Service{
		rt:      rt,
		b:       b,
		msgChan: msgChan,
	}
}

func (s *Service) Start() <-chan error {
	e := make(chan error)
	go s.start(e)
	return e
}

func (s *Service) start(e chan error) {
	go func(msgChan <-chan p2p.Message) {
		msg := <-msgChan
		msgType := msg.GetType()
		if msgType == p2p.TransactionMsgType {
			// process tx
		}
	}(s.msgChan)

	e <- nil
}

func (s *Service) Stop() <-chan error {
	e := make(chan error)

	return e
}

// ProcessTransaction attempts to validates the transaction
// if it is validated, it is added to the transaction pool of the BABE session
func (s *Service) ProcessTransaction(e core.Extrinsic) error {
	validity, err := s.validateTransaction(e)
	if err != nil {
		log.Error("ProcessTransaction", "error", err)
		return err
	}

	vtx := tx.NewValidTransaction(e, validity)
	s.b.PushToTxQueue(vtx)

	return nil
}

// ProcessBlock attempts to add a block to the chain by calling `core_execute_block`
// if the block is validated, it is stored in the block DB and becomes part of the canonical chain
func (s *Service) ProcessBlock(b *core.BlockHeader) {
	return
}

// runs the extrinsic through runtime function TaggedTransactionQueue_validate_transaction
// and returns *Validity
func (s *Service) validateTransaction(e core.Extrinsic) (*tx.Validity, error) {
	ret, err := s.rt.Exec("TaggedTransactionQueue_validate_transaction", 1, 0)
	if err != nil {
		return nil, err
	}

	v := new(tx.Validity)
	_, err = scale.Decode(ret, v)
	return v, err
}
