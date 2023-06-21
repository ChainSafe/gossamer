// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSharedDataGenericReadWrite(t *testing.T) {
	sharedData := NewSharedData("hello world")

	go func(sharedData *SharedData[string]) {
		sharedData.Write("goodbye")
	}(sharedData)

	data := sharedData.Read()

	require.Equal(t, "hello world", data)

	time.Sleep(3 * time.Second)

	data = sharedData.Read()

	require.Equal(t, "goodbye", data)
}

func TestSharedDataGenericAcquireRelease(t *testing.T) {
	sharedData := NewSharedData("hello world")

	sharedData.Acquire()

	go func(sharedData *SharedData[string]) {
		sharedData.Acquire()
		sharedData.inner = "goodbye"
		sharedData.Release()
	}(sharedData)

	data := sharedData.inner

	require.Equal(t, "hello world", data)

	sharedData.Release()

	time.Sleep(3 * time.Second)

	data = sharedData.Read()

	require.Equal(t, "goodbye", data)
}

func TestSharedDataGeneric(t *testing.T) {
	sharedData := NewSharedData("hello world")
	sharedData.Acquire()

	go func(sharedData *SharedData[string]) {
		// This will need to wait for the lock to be released before it can access the data.
		sharedData.Acquire()
		sharedData.inner += "1"
		sharedData.Release()
	}(sharedData)

	require.Equal(t, "hello world", sharedData.inner)

	sharedData.inner += "3"

	require.Equal(t, "hello world3", sharedData.inner)

	sharedData.Release()

	time.Sleep(1 * time.Second)

	sharedData.Acquire()
	sharedData.inner += "2"
	sharedData.Release()

	data := sharedData.Read()
	require.True(t, data == "hello world312")

}
