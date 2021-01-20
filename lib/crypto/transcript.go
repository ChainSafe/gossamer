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
	Bytes []byte 
	Uint64 *uint64
}

// MakeTranscript creates a new *merlin.Transcript from the given VRFTranscriptData
func MakeTranscript(data *VRFTranscriptData) *merlin.Transcript {
	t := merlin.NewTranscript(data.Label)

	for label, val := range data.Items {
		if val.Bytes != nil {
			t.AppendMessage([]byte(label), val.Bytes)
		} else if val.Uint64 != nil {
			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, *val.Uint64)
			t.AppendMessage([]byte(label), buf)
		} else {
			panic("invalid VRFTranscriptValue")
		}
	}

	return t
}