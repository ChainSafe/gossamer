package subscription

import (
	"errors"
	"fmt"
	"strconv"
)

// RPC methods
const (
	authorSubmitAndWatchExtrinsic  string = "author_submitAndWatchExtrinsic"
	chainSubscribeNewHeads         string = "chain_subscribeNewHeads"
	chainSubscribeNewHead          string = "chain_subscribeNewHead"
	chainSubscribeFinalizedHeads   string = "chain_subscribeFinalizedHeads"
	chainSubscribeAllHeads         string = "chain_subscribeAllHeads"
	stateSubscribeStorage          string = "state_subscribeStorage"
	stateSubscribeRuntimeVersion   string = "state_subscribeRuntimeVersion"
	grandpaSubscribeJustifications string = "grandpa_subscribeJustifications"
)

type setupListener func(reqid float64, params interface{}) (Listener, error)

var (
	errUknownParamSubscribeID = errors.New("invalid params format type")
	errCannotParseID          = errors.New("could not parse param id")
	errCannotFindListener     = errors.New("could not find listener")
	errCannotFindUnsubsriber  = errors.New("could not find unsubsriber function")
)

func (c *WSConn) getSetupListener(method string) setupListener {
	switch method {
	case authorSubmitAndWatchExtrinsic:
		return c.initExtrinsicWatch
	case chainSubscribeNewHeads, chainSubscribeNewHead:
		return c.initBlockListener
	case stateSubscribeStorage:
		return c.initStorageChangeListener
	case chainSubscribeFinalizedHeads:
		return c.initBlockFinalizedListener
	case chainSubscribeAllHeads:
		return c.initAllBlocksListerner
	case stateSubscribeRuntimeVersion:
		return c.initRuntimeVersionListener
	case grandpaSubscribeJustifications:
		return c.initGrandpaJustificationListener
	default:
		return nil
	}
}

func (c *WSConn) getUnsubListener(params interface{}) (Listener, error) {
	subscribeID, err := parseSubscribeID(params)
	if err != nil {
		return nil, err
	}

	listener, ok := c.Subscriptions[subscribeID]
	if !ok {
		return nil, fmt.Errorf("subscriber id %v: %w", subscribeID, errCannotFindListener)
	}

	return listener, nil
}

func parseSubscribeID(p interface{}) (uint32, error) {
	switch v := p.(type) {
	case []interface{}:
		if len(v) == 0 {
			return 0, errUknownParamSubscribeID
		}
	default:
		return 0, errUknownParamSubscribeID
	}

	var id uint32
	switch v := p.([]interface{})[0].(type) {
	case float64:
		id = uint32(v)
	case string:
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, errCannotParseID
		}
		id = uint32(i)
	default:
		return 0, errUknownParamSubscribeID
	}

	return id, nil
}
