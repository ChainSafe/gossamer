package parachain

import "github.com/ChainSafe/gossamer/pkg/scale"

// All network messages on the collation peer-set.
type CollationProtocol scale.VaryingDataType

func NewCollationProtocol() CollationProtocol {
	vdt := scale.MustNewVaryingDataType(NewCollatorProtocolMessage())
	return CollationProtocol(vdt)
}

func (c *CollationProtocol) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = CollationProtocol(vdt)
	return
}

func (c *CollationProtocol) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
}

