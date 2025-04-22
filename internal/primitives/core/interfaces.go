// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import "github.com/ChainSafe/gossamer/internal/primitives/externalities"

type CodeExecutor interface {
	/// Call a given method in the runtime.
	/// Returns a tuple of the result (either the output data or an execution error) together with a
	/// `bool`, which is true if native execution was used.
	Call(
		ext externalities.Externalities,
		runtimeCode RuntimeCode,
		method string,
		data []byte,
		context CallContext,
	) (result []byte, native bool, err error)
}
