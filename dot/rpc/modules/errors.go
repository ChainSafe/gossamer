package modules

import "errors"

// ErrSubscriptionTransport error sent when trying to access websocket subscriptions via http
var ErrSubscriptionTransport = errors.New("subscriptions are not available on this transport")
