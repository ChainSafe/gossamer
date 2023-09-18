// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/services"
)

// service can be started and stopped.
type service interface {
	Start() error
	Stop() error
}

// ServiceRegisterer can register a service interface, start or stop all services,
// and get a particular service.
type ServiceRegisterer interface {
	RegisterService(service services.Service)
	StartAll()
	StopAll()
	Get(srvc interface{}) services.Service
}

// BlockJustificationVerifier has a verification method for block justifications.
type BlockJustificationVerifier interface {
	VerifyBlockJustification(common.Hash, []byte) error
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}
