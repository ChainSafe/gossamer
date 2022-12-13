// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memory

import (
	"context"
	"sync"
	"testing"
	"time"
)

func Test_Database_threadSafety(t *testing.T) {
	// This test consists in checking for concurrent access
	// using the -race detector.
	t.Parallel()

	var startWg, endWg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	const operations = 5
	const parallelism = 3
	const goroutines = parallelism * operations
	startWg.Add(goroutines)
	endWg.Add(goroutines)

	const testDuration = 50 * time.Millisecond
	go func() {
		timer := time.NewTimer(time.Hour)
		startWg.Wait()
		_ = timer.Reset(testDuration)
		<-timer.C
		cancel()
	}()

	runInLoop := func(f func()) {
		defer endWg.Done()
		startWg.Done()
		startWg.Wait()
		for ctx.Err() == nil {
			f()
		}
	}

	database := New()
	key := []byte{1}
	value := []byte{2}

	for i := 0; i < parallelism; i++ {
		go runInLoop(func() {
			_, _ = database.Get(key)
		})

		go runInLoop(func() {
			_ = database.Set(key, value)
		})

		go runInLoop(func() {
			_ = database.Delete(key)
		})

		go runInLoop(func() {
			batch := database.NewWriteBatch()
			_ = batch.Set(key, value)
			_ = batch.Delete(key)
			_ = batch.Set(key, value)
			_ = batch.Flush()
		})
		go runInLoop(func() {
			_ = database.DropAll()
		})
	}

	endWg.Wait()
}
