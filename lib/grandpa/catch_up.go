package grandpa

import (
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"sync"
	"sync/atomic"
	"time"
)

type catchUp struct {
	lock sync.Mutex

	grandpa         *Service
	readyToCatchUp  *atomic.Bool
	shutdownCatchup chan struct{}
}

func newCatchUp(grandpa *Service) *catchUp {
	c := &catchUp{
		readyToCatchUp:  &atomic.Bool{},
		grandpa:         grandpa,
		shutdownCatchup: make(chan struct{}),
	}
	c.readyToCatchUp.Store(true)
	return c
}

func (c *catchUp) tryCatchUp(round uint64, setID uint64, peer peer.ID) error {
	logger.Warnf("Trying to catch up")
	c.lock.Lock()
	if !c.readyToCatchUp.Load() {
		// Fine we just skip
		return nil
	}
	c.lock.Lock()
	catchUpRequest := newCatchUpRequest(round, setID)
	cm, err := catchUpRequest.ToConsensusMessage()
	if err != nil {
		return fmt.Errorf("converting catchUpRequest to network message: %w", err)
	}

	logger.Warnf("sending catchup request message: %v", catchUpRequest)
	err = c.grandpa.network.SendMessage(peer, cm)
	if err != nil {
		return fmt.Errorf("sending catchUpRequest to network message: %w", err)
	}
	c.readyToCatchUp.Store(false)
	logger.Warnf("successfully tryed catch up")
	return nil
}

func (c *catchUp) initCatchUp() {
	logger.Warnf("Initializing catch up")
	const duration = time.Second * 30
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Warnf("Ready to catch up again")
			c.readyToCatchUp.Store(true)
		case <-c.shutdownCatchup:
			logger.Warnf("Closing catch up")
			return
		}
	}
}
