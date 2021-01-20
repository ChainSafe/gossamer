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
package crypto

import (
	"encoding/binary"

	"github.com/gtank/merlin"
)

// VRFTranscriptData represents data used to construct a VRF transcript
type VRFTranscriptData struct {
	Label string
	Items map[string]*VRFTranscriptValue
}

// VRFTranscriptValue represents a value to be added to a transcript
type VRFTranscriptValue struct { // TODO: turn this into a variadic type
	Bytes  []byte
	Uint64 *uint64
}

// MakeTranscript creates a new *merlin.Transcript from the given VRFTranscriptData
func MakeTranscript(data *VRFTranscriptData) *merlin.Transcript {
	t := merlin.NewTranscript(data.Label)

	for label, val := range data.Items {
		if val.Bytes != nil {
			t.AppendMessage([]byte(label), val.Bytes)
		} else if val.Uint64 != nil {
			AppendUint64(t, []byte(label), *val.Uint64)
		} else {
			panic("invalid VRFTranscriptValue")
		}
	}

	return t
}

// AppendUint64 appends a uint64 to the given transcript using the given label
func AppendUint64(t *merlin.Transcript, label []byte, n uint64) *merlin.Transcript {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, n)
	t.AppendMessage(label, buf)
	return t
}
