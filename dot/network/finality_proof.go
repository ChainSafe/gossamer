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

package network

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// FinalityProofRequest ...
type FinalityProofRequest struct {
	Hash    common.Hash
	Request []byte
}

// SubProtocol returns the finality-proof sub-protocol.
func (f *FinalityProofRequest) SubProtocol() string {
	return finalityProofID
}

// Decode the message into a FinalityProofRequest.
func (f *FinalityProofRequest) Decode(b []byte) error {
	msg, err := scale.Decode(b, f)
	if err != nil {
		return err
	}

	f.Hash = msg.(*FinalityProofRequest).Hash
	f.Request = msg.(*FinalityProofRequest).Request

	return nil
}

// Encode a FinalityProofRequest Msg Type containing the FinalityProofRequest using scale.Encode.
func (f *FinalityProofRequest) Encode() ([]byte, error) {
	return scale.Encode(f)
}

// String implements the Stringer interface.
func (f *FinalityProofRequest) String() string {
	return fmt.Sprintf("FinalityProofRequest Hash=%s Request=%x", f.Hash, f.Request)
}

// FinalityProofResponse ...
type FinalityProofResponse struct {
	Proof []byte
}

// SubProtocol returns the finality-proof sub-protocol.
func (f *FinalityProofResponse) SubProtocol() string {
	return finalityProofID
}

// Decode the message into a FinalityProofResponse.
func (f *FinalityProofResponse) Decode(b []byte) error {
	msg, err := scale.Decode(b, f)
	if err != nil {
		return err
	}

	f.Proof = msg.(*FinalityProofResponse).Proof

	return nil
}

// Encode a FinalityProofResponse Msg Type containing the FinalityProofResponse using scale.Encode.
func (f *FinalityProofResponse) Encode() ([]byte, error) {
	return scale.Encode(f)
}

// String implements the Stringer interface.
func (f *FinalityProofResponse) String() string {
	return fmt.Sprintf("FinalityProofResponse Proof=%x", f.Proof)
}
