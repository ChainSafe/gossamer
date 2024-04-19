// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidate_validation

func Register(overseerChan chan<- any) (*CandidateValidation, error) {
	candidateValidation := CandidateValidation{
		SubSystemToOverseer: overseerChan,
	}

	return &candidateValidation, nil
}
