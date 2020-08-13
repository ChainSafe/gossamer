// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"github.com/ChainSafe/gossamer/dot/network"
)

// processConsensusMessage routes a consensus message from the network to the finality gadget
func (s *Service) processConsensusMessage(msg *network.ConsensusMessage) error {
	return s.consensusMessageHandler.HandleMessage(msg)
}

// sendVoteMessages routes a VoteMessage from the finality gadget to the network
func (s *Service) sendVoteMessages() {
	out := s.finalityGadget.GetVoteOutChannel()

	for {
		select {
		case v := <-out:
			msg, err := v.ToConsensusMessage()
			if err != nil {
				s.logger.Error("failed to convert VoteMessage to ConsensusMessage", "msg", msg)
				continue
			}

			s.logger.Debug("sending VoteMessage to network", "msg", msg)
			s.safeMsgSend(msg)
		case <-s.ctx.Done():
			return
		}
	}
}

// sendFinalityMessages routes a FinalizationMessage from the finality gadget to the network
func (s *Service) sendFinalizationMessages() {
	out := s.finalityGadget.GetFinalizedChannel()

	for {
		select {
		case v := <-out:
			if v == nil {
				continue
			}

			s.logger.Info("finalized block!!!", "msg", v)
			msg, err := v.ToConsensusMessage()
			if err != nil {
				s.logger.Error("failed to convert FinalizationMessage to ConsensusMessage", "msg", msg)
				continue
			}

			s.logger.Debug("sending FinalityMessage to network", "msg", v)
			s.safeMsgSend(msg)
		case <-s.ctx.Done():
			return
		}
	}
}
