// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package keystore

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/stretchr/testify/require"
)

func TestNewKeyring(t *testing.T) {
	kr, err := NewKeyring()
	require.Nil(t, err)

	v := reflect.ValueOf(kr).Elem()
	for i := 0; i < v.NumField(); i++ {
		key := v.Field(i).Interface().(*sr25519.Keypair).Private().Hex()
		if key != privateKeys[i] {
			t.Fatalf("Fail: got %s expected %s", key, privateKeys[i])
		}
	}
}
