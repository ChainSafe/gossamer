package subscription

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
)

type setupListener func(reqid float64, params interface{}) (Listener, error)

func (c *WSConn) getSetupListener(method string) setupListener {
	c.qtyListeners++

	switch method {
	case "chain_subscribeNewHeads", "chain_subscribeNewHead":
		return c.initBlockListener
		// case "state_subscribeStorage":
		// 	_, err2 := c.initStorageChangeListener(reqid, params)
		// 	if err2 != nil {
		// 		logger.Warn("failed to create state change listener", "error", err2)
		// 		continue
		// 	}

		// case "chain_subscribeFinalizedHeads":
		// 	bfl, err3 := c.initBlockFinalizedListener(reqid)
		// 	if err3 != nil {
		// 		logger.Warn("failed to create block finalised", "error", err3)
		// 		continue
		// 	}
		// 	c.startListener(bfl)
		// case "state_subscribeRuntimeVersion":
		// 	rvl, err4 := c.initRuntimeVersionListener(reqid)
		// 	if err4 != nil {
		// 		logger.Warn("failed to create runtime version listener", "error", err4)
		// 		continue
		// 	}
		// 	c.startListener(rvl)
		// case "state_unsubscribeStorage":
		// 	c.unsubscribeStorageListener(reqid, params)

	}

	return nil
}

func initBlockListener(c *WSConn, reqID float64) (uint, error) {
	bl := &BlockListener{
		Channel: make(chan *types.Block),
		wsconn:  c,
	}

	if c.BlockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return 0, fmt.Errorf("error BlockAPI not set")
	}

	chanID, err := c.BlockAPI.RegisterImportedChannel(bl.Channel)
	if err != nil {
		return 0, err
	}
	bl.ChanID = chanID
	c.qtyListeners++
	bl.subID = c.qtyListeners
	c.Subscriptions[bl.subID] = bl
	c.BlockSubChannels[bl.subID] = chanID
	initRes := NewSubscriptionResponseJSON(bl.subID, reqID)
	c.safeSend(initRes)

	return bl.subID, nil
}
