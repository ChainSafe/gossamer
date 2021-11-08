// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"errors"

	"github.com/gorilla/rpc/v2/json2"
)

// ErrCannotValidateTx is returned if the call to runtime function TaggedTransactionQueueValidateTransaction fails
var ErrCannotValidateTx = errors.New("could not validate transaction")

// ErrInvalidTransaction is returned if the call to runtime function TaggedTransactionQueueValidateTransaction fails with
//  value of [1, 0, x]
var ErrInvalidTransaction = &json2.Error{Code: 1010, Message: "Invalid Transaction"}

// ErrUnknownTransaction is returned if the call to runtime function TaggedTransactionQueueValidateTransaction fails with
//  value of [1, 1, x]
var ErrUnknownTransaction = &json2.Error{Code: 1011, Message: "Unknown Transaction Validity"}

// ErrNilStorage is returned when the runtime context storage isn't set
var ErrNilStorage = errors.New("runtime context storage is nil")
