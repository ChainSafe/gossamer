package messages

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// CollationProtocol represents all network messages on the collation peer-set.
type CollationProtocol scale.VaryingDataType

// NewCollationProtocol returns a new collation protocol varying data type
func NewCollationProtocol() CollationProtocol {
	vdt := scale.MustNewVaryingDataType(NewCollatorProtocolMessage())
	return CollationProtocol(vdt)
}

// New will enable scale to create new instance when needed
func (CollationProtocol) New() CollationProtocol {
	return NewCollationProtocol()
}

// Set will set a value using the underlying  varying data type
func (c *CollationProtocol) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = CollationProtocol(vdt)
	return
}

// Value returns the value from the underlying varying data type
func (c *CollationProtocol) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
}

// CollatorProtocolMessage represents Network messages used by the collator protocol subsystem
type CollatorProtocolMessage scale.VaryingDataType

// Index returns the index of varying data type
func (CollatorProtocolMessage) Index() uint {
	return 0
}

// NewCollatorProtocolMessage returns a new collator protocol message varying data type
func NewCollatorProtocolMessage() CollatorProtocolMessage {
	vdt := scale.MustNewVaryingDataType(Declare{}, AdvertiseCollation{}, CollationSeconded{})
	return CollatorProtocolMessage(vdt)
}

// New will enable scale to create new instance when needed
func (CollatorProtocolMessage) New() CollatorProtocolMessage {
	return NewCollatorProtocolMessage()
}

// Set will set a value using the underlying  varying data type
func (c *CollatorProtocolMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = CollatorProtocolMessage(vdt)
	return
}

// Value returns the value from the underlying varying data type
func (c *CollatorProtocolMessage) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
}

// Declare the intent to advertise collations under a collator ID, attaching a
// signature of the `PeerId` of the node using the given collator ID key.
type Declare struct {
	CollatorId        parachaintypes.CollatorID        `scale:"1"`
	ParaID            uint32                           `scale:"2"`
	CollatorSignature parachaintypes.CollatorSignature `scale:"3"`
}

// Index returns the index of varying data type
func (Declare) Index() uint {
	return 0
}

// AdvertiseCollation contains a relay parent hash and is used to advertise a collation to a validator.
// This will only advertise a collation if there exists one for the given relay parent and the given peer is
// set as validator for our para at the given relay parent.
// It can only be sent once the peer has declared that they are a collator with given ID
type AdvertiseCollation common.Hash

// Index returns the index of varying data type
func (AdvertiseCollation) Index() uint {
	return 1
}

// CollationSeconded represents that a collation sent to a validator was seconded.
type CollationSeconded struct {
	RelayParent common.Hash                                 `scale:"1"`
	Statement   parachaintypes.UncheckedSignedFullStatement `scale:"2"`
}

// Index returns the index of varying data type
func (CollationSeconded) Index() uint {
	return 4
}

const MaxCollationMessageSize uint64 = 100 * 1024

// Type returns CollationMsgType
func (CollationProtocol) Type() network.MessageType {
	return network.CollationMsgType
}

// Hash returns the hash of the CollationProtocolV1
func (cp CollationProtocol) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := cp.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (cp CollationProtocol) Encode() ([]byte, error) {
	enc, err := scale.Marshal(cp)
	if err != nil {
		return nil, err
	}
	return enc, nil
}
