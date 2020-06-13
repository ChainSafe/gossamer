package core

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"

	"github.com/stretchr/testify/require"
)

func TestProcessConsensusMessage(t *testing.T) {
	fg := &mockFinalityGadget{
		in:        make(chan FinalityMessage, 2),
		out:       make(chan FinalityMessage, 2),
		finalized: make(chan FinalityMessage, 2),
	}

	s := NewTestService(t, &Config{
		FinalityGadget: fg,
	})
	err := s.processConsensusMessage(testConsensusMessage)
	require.NoError(t, err)

	select {
	case f := <-fg.in:
		require.Equal(t, &mockFinalityMessage{}, f)
	case <-time.After(testMessageTimeout):
		t.Fatal("did not receive finality message")
	}
}

func TestSendVoteMessages(t *testing.T) {
	fg := &mockFinalityGadget{
		in:        make(chan FinalityMessage, 2),
		out:       make(chan FinalityMessage, 2),
		finalized: make(chan FinalityMessage, 2),
	}

	msgSend := make(chan network.Message, 2)

	s := NewTestService(t, &Config{
		MsgSend:        msgSend,
		FinalityGadget: fg,
	})

	go s.sendVoteMessages()
	fg.out <- &mockFinalityMessage{}

	select {
	case msg := <-msgSend:
		require.Equal(t, testConsensusMessage, msg)
	case <-time.After(testMessageTimeout):
		t.Fatal("did not receive finality message")
	}
}

func TestSendFinalizationMessages(t *testing.T) {
	fg := &mockFinalityGadget{
		in:        make(chan FinalityMessage, 2),
		out:       make(chan FinalityMessage, 2),
		finalized: make(chan FinalityMessage, 2),
	}

	msgSend := make(chan network.Message, 2)

	s := NewTestService(t, &Config{
		MsgSend:        msgSend,
		FinalityGadget: fg,
	})

	go s.sendFinalizationMessages()
	fg.finalized <- &mockFinalityMessage{}

	select {
	case msg := <-msgSend:
		require.Equal(t, testConsensusMessage, msg)
	case <-time.After(testMessageTimeout):
		t.Fatal("did not receive finality message")
	}
}
