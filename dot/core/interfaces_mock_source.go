package core

import "encoding/json"

type telemetry interface {
	SendMessage(msg json.Marshaler)
}
