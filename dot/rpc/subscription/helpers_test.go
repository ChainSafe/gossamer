// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package subscription

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func setupWSConn(t *testing.T) (*WSConn, *websocket.Conn, func()) {
	t.Helper()

	wskt := new(WSConn)
	var up = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Use a channel to notify when the WebSocket connection is ready
	connReady := make(chan struct{})

	h := func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			log.Print("error while setup handler:", err)
			return
		}

		wskt.Wsconn = c

		// Notify that the WebSocket connection is ready
		close(connReady)
	}

	server := httptest.NewServer(http.HandlerFunc(h))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, r, err := websocket.DefaultDialer.Dial(wsURL, nil)
	r.Body.Close()

	require.NoError(t, err)

	// Wait for the WebSocket connection to be ready before proceeding
	<-connReady
	require.NotNil(t, wskt.Wsconn)

	cancel := func() {
		server.Close()
		ws.Close()
		wskt.Wsconn.Close()
	}

	return wskt, ws, cancel
}
