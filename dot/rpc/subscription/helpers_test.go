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

	h := func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			log.Print("error while setup handler:", err)
			return
		}

		wskt.Wsconn = c
	}

	server := httptest.NewServer(http.HandlerFunc(h))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, r, err := websocket.DefaultDialer.Dial(wsURL, nil)
	defer r.Body.Close()

	require.NoError(t, err)

	cancel := func() {
		server.Close()
		ws.Close()
		wskt.Wsconn.Close()
	}

	return wskt, ws, cancel
}
