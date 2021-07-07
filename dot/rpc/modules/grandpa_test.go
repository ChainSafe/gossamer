// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package modules

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
)

func TestGrandpaProveFinality(t *testing.T) {
	testStateService := newTestStateService(t)

	state.AddBlocksToState(t, testStateService.Block, 3)
	bestBlock, err := testStateService.Block.BestBlock()

	if err != nil {
		t.Errorf("Fail: bestblock failed")
	}

	gmSvc := NewGrandpaModule(testStateService.Block)

	testStateService.Block.SetJustification(bestBlock.Header.ParentHash, make([]byte, 10))
	testStateService.Block.SetJustification(bestBlock.Header.Hash(), make([]byte, 11))

	var expectedResponse ProveFinalityResponse
	expectedResponse = append(expectedResponse, make([]byte, 10), make([]byte, 11))

	res := new(ProveFinalityResponse)
	err = gmSvc.ProveFinality(nil, &ProveFinalityRequest{
		blockHashStart: bestBlock.Header.ParentHash,
		blockHashEnd:   bestBlock.Header.Hash(),
	}, res)

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*res, expectedResponse) {
		t.Errorf("Fail: expected: %+v got: %+v\n", res, &expectedResponse)
	}
}
