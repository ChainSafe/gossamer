package tel2

import (
	"encoding/json"
	"fmt"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

type telemetryConnection struct {
	wsconn *websocket.Conn
	verbosity int
}

type telemetryMessage struct {
	values map[string]interface{}
}

// Handler struct for holding telemetry related things
type telHandler struct {
	msg chan telemetryMessage
	connections []telemetryConnection
}

type keyValue struct {
	key string
	value interface{}
}
var (
	once            sync.Once
	handlerInstance *telHandler
)

// GetInstance singleton pattern to for accessing TelemetryHandler
func GetTelInstance() *telHandler {
	if handlerInstance == nil {
		once.Do(
			func() {
				handlerInstance = &telHandler{
					msg: make(chan telemetryMessage, 3),
				}
				go handlerInstance.startListening()
			})
	}
	return handlerInstance
}

func NewTelemetryMessage(values ...keyValue) *telemetryMessage {
	mvals := make(map[string]interface{})
	for i, v := range values {
		fmt.Printf("Key %v %v\n", i, v)
		mvals[v.key] = v.value
	}
	return &telemetryMessage{
		values: mvals,
	}
}

func NewKeyValue(key string,  value interface{}) keyValue {
	return keyValue{
		key:  key,
		value: value,
	}
}

func (t *telHandler) AddConnections(conns []*genesis.TelemetryEndpoint) {
	for _, v := range conns {
		c, _, err := websocket.DefaultDialer.Dial(v.Endpoint, nil)
		if err != nil {
			// todo (ed) try reconnecting if there is an error connecting
			fmt.Printf("Error %v\n", err)
			continue
		}
		tConn := telemetryConnection{
			wsconn:    c,
			verbosity: v.Verbosity,
		}
		t.connections = append(t.connections, tConn)
	}
}

func (t *telHandler) SendMessage(msg *telemetryMessage) {
	t.msg <- *msg
}

func (t *telHandler) startListening() {
	for {
		msg := <- t.msg
		for _, v := range t.connections {
			err := v.wsconn.WriteMessage(websocket.TextMessage, msgToBytes(msg))
			if err != nil {
				// TODO (ed) determine how to handle this error
				fmt.Printf("ERROR connecting to telemetry %v\n", err)
			}
			fmt.Printf("Send to conn %v msg %s\n", v.wsconn.RemoteAddr(), msgToBytes(msg) )
		}
	}
}

func msgToBytes(message telemetryMessage) []byte {
	res := make(map[string]interface{})
	res["id"] = 1 // todo determine how this is used
	res["payload"] =  message.values
	res["ts"] = time.Now()
	resB, err := json.Marshal(res)
	if err != nil {
		return nil
	}
	return resB
}
