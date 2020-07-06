package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/scale"
)

// MessageHandler handles GRANDPA consensus messages
type MessageHandler struct {
	grandpa    *Service
	blockState BlockState
}

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(grandpa *Service, blockState BlockState) *MessageHandler {
	return &MessageHandler{
		grandpa:    grandpa,
		blockState: blockState,
	}
}

// HandleMessage handles a GRANDPA consensus message
// if it is a FinalizationMessage, it updates the BlockState
// if it is a VoteMessage, it sends it to the GRANDPA service
func (h *MessageHandler) HandleMessage(msg *ConsensusMessage) error {
	m, err := decodeMessage(msg)
	if err != nil {
		return err
	}

	fm, ok := m.(*FinalizationMessage)
	if ok {
		// set finalized head for round in db
		err = h.blockState.SetFinalizedHash(fm.Vote.hash, fm.Round)
		if err != nil {
			return err
		}

		// set latest finalized head in db
		err = h.blockState.SetFinalizedHash(fm.Vote.hash, 0)
		if err != nil {
			return err
		}
	}

	vm, ok := m.(*VoteMessage)
	if h.grandpa != nil && ok {
		// send vote message to grandpa service
		h.grandpa.in <- vm
	}

	return nil
}

// decodeMessage decodes a network-level consensus message into a GRANDPA VoteMessage or FinalizationMessage
func decodeMessage(msg *ConsensusMessage) (m FinalityMessage, err error) {
	var mi interface{}

	switch msg.Data[0] {
	case voteType:
		mi, err = scale.Decode(msg.Data[1:], &VoteMessage{Message: new(SignedMessage)})
		m = mi.(*VoteMessage)
	case finalizationType:
		mi, err = scale.Decode(msg.Data[1:], &FinalizationMessage{})
		m = mi.(*FinalizationMessage)
	default:
		return nil, ErrInvalidMessageType
	}

	if err != nil {
		return nil, err
	}

	return m, nil
}
