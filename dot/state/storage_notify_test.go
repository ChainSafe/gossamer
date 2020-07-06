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
package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStorageState_RegisterStorageChangeChannel(t *testing.T) {
	ss := newTestStorageState(t)

	ch := make(chan *KeyValue, 3)
	id, err := ss.RegisterStorageChangeChannel(ch)
	require.NoError(t, err)

	defer ss.UnregisterStorageChangeChannel(id)

	// three storage change events
	ss.SetStorage([]byte("mackcom"), []byte("wuz here"))
	ss.SetStorage([]byte("key1"), []byte("value1"))
	ss.SetStorage([]byte("key1"), []byte("value2"))

	for i := 0; i < 3; i++ {
		select {
		case <-ch:
		case <-time.After(testMessageTimeout):
			t.Fatal("did not receive storage change message")
		}
	}
}
