package parachain

import "github.com/ChainSafe/gossamer/pkg/scale"

// CollationProtocol represents all network messages on the collation peer-set.
type CollationProtocol scale.VaryingDataType

// NewCollationProtocol returns a new collation protocol varying data type
func NewCollationProtocol() CollationProtocol {
	vdt := scale.MustNewVaryingDataType(NewCollatorProtocolMessage())
	return CollationProtocol(vdt)
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
