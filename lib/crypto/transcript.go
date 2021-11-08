// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package crypto

import (
	"encoding/binary"

	"github.com/gtank/merlin"
)

// AppendUint64 appends a uint64 to the given transcript using the given label
func AppendUint64(t *merlin.Transcript, label []byte, n uint64) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, n)
	t.AppendMessage(label, buf)
}
