package subscription

import (
	"errors"
	"fmt"
	"strconv"
)

var errUknownParamSubscribeID = errors.New("invalid params format type")
var errCannotParseID = errors.New("could not parse param id")
var errCannotFindListener = errors.New("could not find listener")
var errCannotFindUnsubsriber = errors.New("could not find unsubsriber function")

type unsubListener func(reqid float64, l Listener, params interface{})
type setupListener func(reqid float64, params interface{}) (Listener, error)

func (c *WSConn) getSetupListener(method string) setupListener {
	switch method {
	case "chain_subscribeNewHeads", "chain_subscribeNewHead":
		return c.initBlockListener
	case "state_subscribeStorage":
		return c.initStorageChangeListener
	case "chain_subscribeFinalizedHeads":
		return c.initBlockFinalizedListener
	case "state_subscribeRuntimeVersion":
		return c.initRuntimeVersionListener
	case "grandpa_subscribeJustifications":
		return c.initGrandpaJustificationListener
	default:
		return nil
	}
}

func (c *WSConn) getUnsubListener(method string, params interface{}) (unsubListener, Listener, error) {
	subscribeID, err := parseSubscribeID(params)
	if err != nil {
		return nil, nil, err
	}

	listener, ok := c.Subscriptions[subscribeID]
	if !ok {
		return nil, nil, fmt.Errorf("subscriber id %v: %w", subscribeID, errCannotFindListener)
	}

	var unsub unsubListener

	switch method {
	case "state_unsubscribeStorage":
		unsub = c.unsubscribeStorageListener
	case "state_unsubscribeRuntimeVersion":
		unsub = c.unsubscribeRuntimeVersionListener
	case "grandpa_unsubscribeJustifications":
		unsub = c.unsubscribeGrandpaJustificationListener
	default:
		return nil, nil, errCannotFindUnsubsriber
	}

	return unsub, listener, nil
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
