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

package optional

import (
	"testing"
)

func TestNewBoolean(t *testing.T) {
	// Non-existant
	testBool := NewBoolean(false, false)

	if testBool.Exists() {
		t.Fatal("exist should be false")
	}
	if testBool.value {
		t.Fatal("value should be false")
	}
}

func TestSet(t *testing.T) {
	testBool := NewBoolean(false, false)

	if testBool.Exists() {
		t.Fatal("exist should be false")
	}
	if testBool.value {
		t.Fatal("value should be false")
	}

	testBool.Set(true, true)

	if !testBool.Exists() {
		t.Fatal("exist should be true")
	}
	if !testBool.value {
		t.Fatal("value should be true")
	}
}

func TestExists(t *testing.T) {
	// Non-existant
	testBool := NewBoolean(false, false)

	if testBool.Exists() {
		t.Fatal("exist should be false")
	}

	// TODO: confirm if setting should set Exists value to true
	testBool.Set(true, false)
	if !testBool.Exists() {
		t.Fatal("exist should be true")
	}
}

func TestValue(t *testing.T) {
	// Non-existant
	testBool := NewBoolean(false, false)

	if testBool.Value() {
		t.Fatal("value should be false")
	}

	testBool.Set(false, true)
	if !testBool.Value() {
		t.Fatal("exist should be true")
	}
}
