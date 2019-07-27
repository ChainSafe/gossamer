package p2p

import (
	leb128 "github.com/filecoin-project/go-leb128"
)

func LEB128ToUint64(in []byte) uint64 {
	return leb128.ToUInt64(in)
}