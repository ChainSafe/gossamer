// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import "sync"

//type SharedAuthoritySet struct {
//	inner SharedData
//}

type SharedData struct {
	inner SharedDataInner
	//condVar bool // condvar is a rust type, investigate. Go sync package has a cond var
	condVar sync.Cond
}

type SharedDataInner struct {
	lock   sync.Mutex
	inner  AuthoritySet
	locked bool
}
