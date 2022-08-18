// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"github.com/gorilla/rpc/v2/json2"
)

// ErrInvalidTransaction is returned if the call to runtime function
// TaggedTransactionQueueValidateTransaction fails with value of [1, 0, x]
var ErrInvalidTransaction = &json2.Error{Code: 1010, Message: "Invalid Transaction"}
