// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"fmt"
)

var (
	// ErrNilBlockState is returned when BlockState is nil
	ErrNilBlockState = errors.New("cannot have nil BlockState")

	// ErrNilStorageState is returned when StorageState is nil
	ErrNilStorageState = errors.New("cannot have nil StorageState")

	// ErrNilKeystore is returned when keystore is nil
	ErrNilKeystore = errors.New("cannot have nil keystore")

	// ErrServiceStopped is returned when the service has been stopped
	ErrServiceStopped = errors.New("service has been stopped")

	// ErrInvalidBlock is returned when a block cannot be verified
	ErrInvalidBlock = errors.New("could not verify block")

	// ErrNilVerifier is returned when trying to instantiate a Syncer without a Verifier
	ErrNilVerifier = errors.New("cannot have nil Verifier")

	// ErrNilRuntime is returned when trying to instantiate a Service or Syncer without a runtime
	ErrNilRuntime = errors.New("cannot have nil runtime")

	// ErrNilBlockProducer is returned when trying to instantiate a block producing Service without a block producer
	ErrNilBlockProducer = errors.New("cannot have nil BlockProducer")

	// ErrNilConsensusMessageHandler is returned when trying to instantiate a Service without a FinalityMessageHandler
	ErrNilConsensusMessageHandler = errors.New("cannot have nil ErrNilFinalityMessageHandler")

	// ErrNilNetwork is returned when the Network interface is nil
	ErrNilNetwork = errors.New("cannot have nil Network")

	// ErrEmptyRuntimeCode is returned when the storage :code is empty
	ErrEmptyRuntimeCode = errors.New("new :code is empty")

	// ErrNilDigestHandler is returned when the DigestHandler interface is nil
	ErrNilDigestHandler = errors.New("cannot have nil DigestHandler")

	errNilCodeSubstitutedState = errors.New("cannot have nil CodeSubstitutedStat")
)

// ErrNilChannel is returned if a channel is nil
func ErrNilChannel(s string) error {
	return fmt.Errorf("cannot have nil channel %s", s)
}

// ErrMessageCast is returned if unable to cast a network.Message to a type
func ErrMessageCast(s string) error {
	return fmt.Errorf("could not cast network.Message to %s", s)
}

// ErrUnsupportedMsgType is returned if we receive an unknown message type
func ErrUnsupportedMsgType(d byte) error {
	return fmt.Errorf("received unsupported message type %d", d)
}
