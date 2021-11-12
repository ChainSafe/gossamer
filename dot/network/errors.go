package network

import (
	"errors"
)

var (
	errCannotValidateHandshake     = errors.New("failed to validate handshake")
	errInvalidNotificationsMessage = errors.New("message is not NotificationsMessage")
	errMessageIsNotHandshake       = errors.New("failed to convert message to Handshake")
	errMissingHandshakeMutex       = errors.New("outboundHandshakeMutex does not exist")
	errInvalidHandshakeForPeer     = errors.New("peer previously sent invalid handshake")
	errHandshakeTimeout            = errors.New("handshake timeout reached")
)
