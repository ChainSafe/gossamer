// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package modules

import "github.com/ChainSafe/gossamer/lib/common"

func stringToHex(s string) (hex string) {
	return common.BytesToHex([]byte(s))
}

func makeChange(keyHex, valueHex string) [2]*string {
	return [2]*string{&keyHex, &valueHex}
}
