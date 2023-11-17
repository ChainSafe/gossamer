package grandpa

import "github.com/ChainSafe/gossamer/pkg/scale"

/*
	Following is from primitives/consensus/common
*/

// BlockOrigin Block data origin
type BlockOrigin scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (bo *BlockOrigin) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*bo)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*bo = BlockOrigin(vdt)
	return nil
}

// Value will return the value from the underying VaryingDataType
func (bo *BlockOrigin) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*bo)
	return vdt.Value()
}

// NewBlockOrigin is constructor for BlockOrigin
func NewBlockOrigin() BlockOrigin {
	vdt := scale.MustNewVaryingDataType(Genesis{}, NetworkInitialSync{}, NetworkBroadcast{}, ConsensusBroadcast{},
		Own{}, File{})
	return BlockOrigin(vdt)
}

// Genesis block built into the client
type Genesis struct{}

// Index returns the VDT index
func (Genesis) Index() uint { return 0 }

// NetworkInitialSync Block is part of the initial sync with the network
type NetworkInitialSync struct{}

// Index returns the VDT index
func (NetworkInitialSync) Index() uint { return 1 }

// NetworkBroadcast Block was broadcasted on the network
type NetworkBroadcast struct{}

// Index returns the VDT index
func (NetworkBroadcast) Index() uint { return 2 }

// ConsensusBroadcast Block that was received from the network and validated in the consensus process
type ConsensusBroadcast struct{}

// Index returns the VDT index
func (ConsensusBroadcast) Index() uint { return 3 }

// Own Block that was collated by this node
type Own struct{}

// Index returns the VDT index
func (Own) Index() uint { return 4 }

// File Block was imported from a file
type File struct{}

// Index returns the VDT index
func (File) Index() uint { return 5 }
