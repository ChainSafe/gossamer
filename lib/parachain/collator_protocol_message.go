package parachain

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Network messages used by the collator protocol subsystem
type CollatorProtocolMessage scale.VaryingDataType

func (c CollatorProtocolMessage) Index() uint {
	return 0
}

func NewCollatorProtocolMessage() CollatorProtocolMessage {
	vdt := scale.MustNewVaryingDataType(Declare{}, AdvertiseCollation{}, CollationSeconded{})
	return CollatorProtocolMessage(vdt)
}

func (c *CollatorProtocolMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = CollatorProtocolMessage(vdt)
	return
}

func (c *CollatorProtocolMessage) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
}

// Declare the intent to advertise collations under a collator ID, attaching a
// signature of the `PeerId` of the node using the given collator ID key.
type Declare struct {
	CollatorId        CollatorID        `scale:"1"`
	ParaId            uint32            `scale:"2"`
	CollatorSignature CollatorSignature `scale:"3"`
}

func (d Declare) Index() uint {
	return 0
}

// Advertise a collation to a validator. Can only be sent once the peer has
// declared that they are a collator with given ID.
type AdvertiseCollation common.Hash

func (a AdvertiseCollation) Index() uint {
	return 1
}

// A collation sent to a validator was seconded.
type CollationSeconded struct {
	Hash                         common.Hash                  `scale:"1"`
	UncheckedSignedFullStatement UncheckedSignedFullStatement `scale:"2"`
}

func (c CollationSeconded) Index() uint {
	return 4
}
