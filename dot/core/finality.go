package core

import (
	"github.com/ChainSafe/gossamer/dot/network"

	log "github.com/ChainSafe/log15"
)

// processConsensusMessage routes a consensus message from the network to the finality gadget
func (s *Service) processConsensusMessage(msg *network.ConsensusMessage) error {
	in := s.finalityGadget.GetVoteInChannel()
	fm, err := s.finalityGadget.DecodeMessage(msg)
	if err != nil {
		return err
	}

	// TODO: safety
	log.Debug("[core] sending VoteMessage to network", "msg", msg)
	in <- fm
	return nil
}

// sendVoteMessages routes a VoteMessage from the finality gadget to the network
func (s *Service) sendVoteMessages() error {
	out := s.finalityGadget.GetVoteOutChannel()
	for v := range out {
		// TODO: safety
		msg, err := v.ToConsensusMessage()
		if err != nil {
			return err
		}
		log.Debug("[core] sending VoteMessage to grandpa", "msg", msg)
		s.msgSend <- msg
	}
	return nil
}

// sendFinalityMessages routes a FinalityMessage from the finality gadget to the network
func (s *Service) sendFinalityMessages() error {
	out := s.finalityGadget.GetFinalizedChannel()
	for v := range out {
		// TODO: safety
		// TODO: update state.finalizedHead
		log.Debug("[core] sending FinalityMessage to network", "msg", v)
		log.Info("[core] finalized block!!!", "msg", v)
		msg, err := v.ToConsensusMessage()
		if err != nil {
			return err
		}

		s.msgSend <- msg
	}
	return nil
}
