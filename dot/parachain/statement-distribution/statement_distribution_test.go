// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

// //nolint
func createStatementDistribution() (*StatementDistribution, chan any) {

	subSystemToOverseer := make(chan any, 10)

	return &StatementDistribution{
		SubSystemToOverseer: subSystemToOverseer,
	}, subSystemToOverseer
}
