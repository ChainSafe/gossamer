// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"
)

//type SharedAuthoritySet struct {
//	inner SharedData
//}

//type SharedData struct {
//	inner SharedDataInner
//	//condVar bool // condvar is a rust type, investigate. Go sync package has a cond var
//	condVar sync.Cond
//}
//
//type SharedDataInner struct {
//	lock   sync.Mutex
//	inner  AuthoritySet
//	locked bool
//}

type SharedData struct {
	inner   SharedDataInner
	condVar *sync.Cond
}

type SharedDataInner struct {
	inner  string
	locked bool
}

func NewSharedData(msg string) *SharedData {
	lock := sync.Mutex{}
	condVar := sync.NewCond(&lock)
	return &SharedData{
		inner: SharedDataInner{
			inner:  msg,
			locked: false,
		},
		condVar: condVar,
	}
}

func (s *SharedData) SharedData() {
	s.condVar.L.Lock()

	fmt.Println("in shared data")
	for s.inner.locked {
		s.condVar.Wait()
	}

	if s.inner.locked {
		panic("wtf")
	}

	fmt.Println("lock acquired")
}

// SharedDataLocked Acquire access to the shared data.
//
// This will give mutable access to the shared data. After the returned mutex guard is dropped,
// the shared data is accessible by other threads. So, this function should be used when
// reading/writing of the shared data in a local context is required.
//
// When requiring to lock shared data for some longer time, even with temporarily releasing the
// lock, [`Self::shared_data_locked`] should be used.
func (s *SharedData) SharedDataLocked() {
	s.condVar.L.Lock()

	for s.inner.locked {
		s.condVar.Wait()
	}

	if s.inner.locked {
		panic("wtf")
	}
	s.inner.locked = true

	fmt.Println("lock acquired")
}
