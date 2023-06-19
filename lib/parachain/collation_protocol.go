package parachain

import "github.com/ChainSafe/gossamer/pkg/scale"

// CollationProtocol represents all network messages on the collation peer-set.
type CollationProtocol scale.VaryingDataType

// NewCollationProtocol returns a new CollationProtocol VaryingDataType
func NewCollationProtocol() CollationProtocol {
	vdt := scale.MustNewVaryingDataType(NewCollatorProtocolMessage())
	return CollationProtocol(vdt)
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (c *CollationProtocol) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = CollationProtocol(vdt)
	return
}

// Value returns the value from the underlying VaryingDataType
func (c *CollationProtocol) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
}
