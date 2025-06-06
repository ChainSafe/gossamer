// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package shareddata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSharedData(t *testing.T) {
	t.Run("shared_data_locking_works", func(t *testing.T) {
		const numThreads = 100
		sharedData := NewSharedData[uint32](0)

		ref, unlock := sharedData.DataMut()

		for i := 0; i < numThreads; i++ {
			go func(i int) {
				ref, unlock := sharedData.DataMut()
				defer unlock()
				// Give the other threads some time to wake up
				time.Sleep(10 * time.Millisecond)
				*ref += 1
			}(i)
		}

		time.Sleep(100 * time.Millisecond)
		// ensure that this is still 0
		require.Equal(t, uint32(0), *ref)
		unlock()

		for {
			sd := sharedData.Data()
			if sd >= numThreads {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

	})

	t.Run("shared_data_locking_works_with_lock", func(t *testing.T) {
		const numThreads = 100
		sharedData := NewSharedData[uint32](0)

		locked := sharedData.Locked()
		ref := locked.MutRef()

		for i := 0; i < numThreads; i++ {
			go func(i int) {
				locked := sharedData.Locked()
				defer locked.Unlock()
				ref := locked.MutRef()
				// Give the other threads some time to wake up
				time.Sleep(10 * time.Millisecond)
				*ref += 1
			}(i)
		}

		time.Sleep(100 * time.Millisecond)
		// ensure that this is still 0
		require.Equal(t, uint32(0), *ref)
		locked.Unlock()

		for {
			sd := sharedData.Data()
			if sd >= numThreads {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

	})
}
