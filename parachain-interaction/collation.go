package parachaininteraction

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// The output of a collator.
//
// This differs from `CandidateCommitments` in two ways:
//
// - does not contain the erasure root; that's computed at the Polkadot level, not at Cumulus
// - contains a proof of validity.
type Collation struct {
	// Messages destined to be interpreted by the Relay chain itself.
	UpwardMessages []upwardMessage `scale:"1"`
	// Horizontal messages sent by the parachain.
	HorizontalMessages []outboundHrmpMessage `scale:"2"`
	// New validation code.
	NewValidationCode *validationCode `scale:"3"`
	// The head-data produced as a result of execution.
	HeadData headData `scale:"4"`
	// Proof to verify the state transition of the parachain.
	ProofOfValidity MaybeCompressedPoV `scale:"5"`
	// The number of messages processed from the DMQ.
	ProcessedDownwardMessages uint32 `scale:"6"`
	// The mark which specifies the block number up to which all inbound HRMP messages are processed.
	HrmpWatermark uint32 `scale:"7"`
}

// upwardMessage is a message from a parachain to its Relay Chain.
type upwardMessage []byte

// outboundHrmpMessage is an HRMP message seen from the perspective of a sender.
type outboundHrmpMessage struct {
	Recipient uint32 `scale:"1"`
	Data      []byte `scale:"2"`
}

// validationCode is Parachain validation code.
type validationCode []byte

// headData is Parachain head data included in the chain.
type headData []byte

type MaybeCompressedPoV scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (mcp *MaybeCompressedPoV) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*mcp)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*mcp = MaybeCompressedPoV(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (mcp *MaybeCompressedPoV) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*mcp)
	return vdt.Value()
}

type raw PoV //skipcq

// Index returns VDT index
func (raw) Index() uint { //skipcq
	return 1
}

func (r raw) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("raw(%s)", PoV(r))
}

type compressed PoV //skipcq

// Index returns VDT index
func (compressed) Index() uint { //skipcq
	return 1
}

func (c compressed) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("raw(%s)", PoV(c))
}

type PoV struct {
	BlockData types.BlockData `scale:"1"`
}
