// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"math"
	"math/big"
)

type voteWeight uint64

type voterWeight uint64

func (vw *voterWeight) checkedAdd(add voterWeight) (err error) {
	sum := new(big.Int).SetUint64(uint64(*vw))
	sum.Add(sum, new(big.Int).SetUint64(uint64(add)))
	if sum.Cmp(new(big.Int).SetUint64(uint64(math.MaxUint64))) > 0 {
		return fmt.Errorf("voterWeight overflow for CheckedAdd")
	}
	*vw = voterWeight(sum.Uint64())
	return nil
}
