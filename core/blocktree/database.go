package blocktree

import (
	"errors"
	"fmt"

	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/common"
)

func (bt *BlockTree) Store() error {
	if bt.db == nil {
		return errors.New("blocktree db is nil")
	}

	enc, err := bt.Encode()
	if err != nil {
		return err
	}

	fmt.Println(enc)

	return bt.db.Put(common.BlockTreeKey, enc)
}

func (bt *BlockTree) Load() error {
	if bt.db == nil {
		return errors.New("blocktree db is nil")
	}

	enc, err := bt.db.Get(common.BlockTreeKey)
	if err != nil {
		return err
	}

	dec, err := scale.Decode(enc, bt)
	bt = dec.(*BlockTree)

	return nil
}

func (bt *BlockTree) Encode() ([]byte, error) {
	return scale.Encode(bt)
}
