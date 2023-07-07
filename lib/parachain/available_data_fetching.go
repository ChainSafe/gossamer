package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// AvailableDataFetchingRequest represents a request to retrieve all available data for a specific candidate.
type AvailableDataFetchingRequest struct {
	// Hash of the candidate for which the available data is requested.
	CandidateHash CandidateHash
}

// Encode returns the SCALE encoding of the AvailableDataFetchingRequest
func (a AvailableDataFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(a)
}

// AvailableDataFetchingResponse represents the possible responses to an available data fetching request.
type AvailableDataFetchingResponse scale.VaryingDataType

// NewAvailableDataFetchingResponse returns a new available data fetching response varying data type
func NewAvailableDataFetchingResponse() AvailableDataFetchingResponse {
	vdt := scale.MustNewVaryingDataType(AvailableData{}, NoSuchData{})
	return AvailableDataFetchingResponse(vdt)
}

// Set will set a value using the underlying  varying data type
func (a *AvailableDataFetchingResponse) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*a)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*a = AvailableDataFetchingResponse(vdt)
	return
}

// Value returns the value from the underlying varying data type
func (a *AvailableDataFetchingResponse) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*a)
	return vdt.Value()
}

// AvailableData represents the data that is kept available for each candidate included in the relay chain.
type AvailableData struct {
	// The Proof-of-Validation (PoV) of the candidate
	PoV Pov `scale:"1"`

	// The persisted validation data needed for approval checks
	ValidationData PersistedValidationData `scale:"2"`
}

// Pov represents a Proof-of-Validity block (PoV block) or a parachain block.
// It contains the necessary data for the parachain specific state transition logic.
type Pov struct {
	BlockData BlockData `scale:"1"`
}

// BlockData represents parachain block data.
// It contains everything required to validate para-block, may contain block and witness data.
type BlockData []byte

// Index returns the index of varying data type
func (AvailableData) Index() uint {
	return 0
}

// PersistedValidationData provides information about how to create the inputs for the validation
// of a candidate by calling the Runtime.
// This information is derived from the parachain state and will vary from parachain to parachain,
// although some of the fields may be the same for every parachain.
type PersistedValidationData struct {
	// The parent head-data
	ParentHead []byte `scale:"1"`

	// The relay-chain block number this is in the context of
	RelayParentNumber parachaintypes.BlockNumber `scale:"2"`

	// The relay-chain block storage root this is in the context of
	RelayParentStorageRoot common.Hash `scale:"3"`

	// The maximum legal size of a POV block, in bytes
	MaxPovSize uint32 `scale:"4"`
}

// NoSuchData indicates that the requested data was not found.
type NoSuchData struct{}

// Index returns the index of varying data type
func (NoSuchData) Index() uint {
	return 1
}

// Encode returns the SCALE encoding of the AvailableDataFetchingResponse
func (a *AvailableDataFetchingResponse) Encode() ([]byte, error) {
	return scale.Marshal(*a)
}

// Decode returns the SCALE decoding of the AvailableDataFetchingResponse.
func (a *AvailableDataFetchingResponse) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, a)
}

// String formats a AvailableDataFetchingResponse as a string
func (p *AvailableDataFetchingResponse) String() string {
	if p == nil {
		return "AvailableDataFetchingResponse=nil"
	}

	v, _ := p.Value()
	availableData, ok := v.(AvailableData)
	if !ok {
		return "AvailableDataFetchingResponse=NoSuchData"
	}
	return fmt.Sprintf("AvailableDataFetchingResponse AvailableData=%+v", availableData)
}
