// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// // Check GRANDPA proof-of-finality for the given block.
// //
// // Returns the vector of headers that MUST be validated + imported
// // AND if at least one of those headers is invalid, all other MUST be considered invalid.
// func checkFinalityProof[
// 	Hash constraints.Ordered,
// 	N constraints.Unsigned,
// 	S comparable,
// 	H Header[Hash, N],
// 	ID AuthorityID,
// ](
// 	currentSetID uint64,
// 	currentAuthorities AuthorityList[ID],
// 	remoteProof []byte,
// ) (FinalityProof[Hash, N, H], error) {
// 	proof := FinalityProof[Hash, N, H]{}
// 	err := scale.Unmarshal(remoteProof, &proof)
// 	if err != nil {
// 		return FinalityProof[Hash, N, H]{}, fmt.Errorf("failed to decode finality proof %s", err)
// 	}

// 	justification := GrandpaJustification[Hash, N, S, ID]{}
// 	err = scale.Unmarshal(proof.Justification, &justification)
// 	if err != nil {
// 		return FinalityProof[Hash, N, H]{}, fmt.Errorf("error decoding justification for header %s", err)
// 	}

// 	err = justification.Verify(currentSetID, currentAuthorities)
// 	if err != nil {
// 		return FinalityProof[Hash, N, H]{}, err
// 	}

// 	return proof, nil
// }

// func createCommit(
// 	t *testing.T,
// 	targetHash string,
// 	targetNum uint,
// 	round uint64,
// 	ID dummyAuthID,
// ) grandpa.Commit[string, uint, string, dummyAuthID] {
// 	t.Helper()
// 	precommit := grandpa.Precommit[string, uint]{
// 		TargetHash:   targetHash,
// 		TargetNumber: targetNum,
// 	}

// 	message := grandpa.Message[string, uint]{
// 		Value: precommit,
// 	}

// 	msg := messageData[string, uint]{
// 		round,
// 		1,
// 		message,
// 	}

// 	encMsg, err := scale.Marshal(msg)
// 	require.NoError(t, err)

// 	signedPrecommit := grandpa.SignedPrecommit[string, uint, string, dummyAuthID]{
// 		Precommit: precommit,
// 		ID:        ID,
// 		Signature: string(encMsg),
// 	}

// 	commit := grandpa.Commit[string, uint, string, dummyAuthID]{
// 		TargetHash:   targetHash,
// 		TargetNumber: targetNum,
// 		Precommits:   []grandpa.SignedPrecommit[string, uint, string, dummyAuthID]{signedPrecommit},
// 	}

// 	return commit
// }

// func TestFinalityProof_FailsIfNoMoreLastFinalizedBlocks(t *testing.T) {
// 	dummyInfo := Info[uint]{
// 		FinalizedNumber: 4,
// 	}
// 	mockBlockchain := NewBlockchainBackendMock[string, uint, testHeader[string, uint]](t)
// 	mockBlockchain.EXPECT().Info().Return(dummyInfo).Once()

// 	mockBackend := NewBackendMock[
// 		string,
// 		uint,
// 		testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]]](t)
// 	mockBackend.EXPECT().Blockchain().Return(mockBlockchain).Once()

// 	// The last finalized block is 4, so we cannot provide further justifications.
// 	authoritySetChanges := AuthoritySetChanges[uint]{}
// 	_, err := proveFinality[
// 		*BackendMock[string, uint, testHeader[string, uint],
// 			*BlockchainBackendMock[string, uint, testHeader[string, uint]]],
// 		string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 		testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]],
// 	](
// 		mockBackend,
// 		authoritySetChanges,
// 		5,
// 		true)
// 	require.ErrorIs(t, err, errBlockNotYetFinalized)
// }

// func TestFinalityProof_IsNoneIfNoJustificationKnown(t *testing.T) {
// 	dummyInfo := Info[uint]{
// 		FinalizedNumber: 4,
// 	}
// 	dummyHash := "dummyHash"
// 	mockBlockchain := NewBlockchainBackendMock[string, uint, testHeader[string, uint]](t)
// 	mockBlockchain.EXPECT().Info().Return(dummyInfo).Once()
// 	mockBlockchain.EXPECT().ExpectBlockHashFromID(uint(4)).Return(dummyHash, nil).Once()
// 	mockBlockchain.EXPECT().Justifications(dummyHash).Return(nil, nil).Once()

// 	mockBackend := NewBackendMock[string, uint, testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]]](t)
// 	mockBackend.EXPECT().Blockchain().Return(mockBlockchain).Times(3)

// 	authoritySetChanges := AuthoritySetChanges[uint]{}
// 	authoritySetChanges.append(0, 4)

// 	// Block 4 is finalized without justification
// 	// => we can't prove finality of 3
// 	proofOf3, err := proveFinality[
// 		*BackendMock[string, uint, testHeader[string, uint],
// 			*BlockchainBackendMock[string, uint, testHeader[string, uint]]],
// 		string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 		testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]],
// 	](
// 		mockBackend,
// 		authoritySetChanges,
// 		3,
// 		true,
// 	)
// 	require.NoError(t, err)
// 	require.Nil(t, proofOf3)
// }

// func TestFinalityProof_CheckFailsWhenProofDecodeFails(t *testing.T) {
// 	// When we can't decode proof from Vec<u8>
// 	authorityList := AuthorityList[dummyAuthID]{}
// 	_, err := checkFinalityProof[string, uint, string, testHeader[string, uint], dummyAuthID](
// 		1,
// 		authorityList,
// 		[]byte{42},
// 	)
// 	require.NotNil(t, err)
// 	require.ErrorContains(t, err, "failed to decode finality proof")
// }

// func TestFinalityProof_CheckFailsWhenProofIsEmpty(t *testing.T) {
// 	// When decoded proof has zero length
// 	authorityList := AuthorityList[dummyAuthID]{}
// 	grandpaJustification := GrandpaJustification[string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 	]{}
// 	encJustification, err := scale.Marshal(grandpaJustification)
// 	require.NoError(t, err)
// 	_, err = checkFinalityProof[string, uint, string, testHeader[string, uint], dummyAuthID](
// 		1,
// 		authorityList,
// 		encJustification,
// 	)
// 	require.NotNil(t, err)
// }

// func TestFinalityProof_CheckFailsWithIncompleteJustification(t *testing.T) {
// 	authorityList := AuthorityList[dummyAuthID]{
// 		Authority[dummyAuthID]{
// 			Key:    dummyAuthID(1),
// 			Weight: uint64(1),
// 		},
// 	}

// 	// Create a commit without precommits
// 	commit := grandpa.Commit[string, uint, string, dummyAuthID]{
// 		TargetHash:   "hash7",
// 		TargetNumber: uint(7),
// 	}

// 	grandpaJust := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Round:  8,
// 		Commit: commit,
// 	}

// 	finalityProof := FinalityProof[string, uint, testHeader[string, uint]]{
// 		Block:         "hash2",
// 		Justification: scale.MustMarshal(grandpaJust),
// 	}

// 	_, err := checkFinalityProof[string, uint, string, testHeader[string, uint], dummyAuthID](
// 		1,
// 		authorityList,
// 		scale.MustMarshal(finalityProof),
// 	)
// 	require.ErrorIs(t, err, errBadJustification)
// }

// func TestFinalityProof_CheckWorksWithCorrectJustification(t *testing.T) {
// 	ID := dummyAuthID(1)
// 	targetHash := "target"
// 	targetNum := uint(21)
// 	authorityList := AuthorityList[dummyAuthID]{
// 		Authority[dummyAuthID]{
// 			Key:    ID,
// 			Weight: uint64(1),
// 		},
// 	}

// 	commit := createCommit(t, targetHash, targetNum, 1, ID)
// 	grandpaJust := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Round:  8,
// 		Commit: commit,
// 	}

// 	finalityProof := FinalityProof[string, uint, testHeader[string, uint]]{
// 		Block:         "hash2",
// 		Justification: scale.MustMarshal(grandpaJust),
// 	}

// 	newFinalityProof, err := checkFinalityProof[string, uint, string, testHeader[string, uint], dummyAuthID](
// 		1,
// 		authorityList,
// 		scale.MustMarshal(finalityProof),
// 	)
// 	require.NoError(t, err)
// 	require.Equal(t, finalityProof, newFinalityProof)
// }

// func TestFinalityProof_UsingAuthoritySetChangesFailsWithUndefinedStart(t *testing.T) {
// 	dummyInfo := Info[uint]{
// 		FinalizedNumber: 8,
// 	}
// 	mockBlockchain := NewBlockchainBackendMock[string, uint, testHeader[string, uint]](t)
// 	mockBlockchain.EXPECT().Info().Return(dummyInfo).Once()

// 	mockBackend := NewBackendMock[string, uint, testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]]](t)
// 	mockBackend.EXPECT().Blockchain().Return(mockBlockchain).Once()

// 	// We are missing the block for the preceding set the start is not well-defined.
// 	authoritySetChanges := AuthoritySetChanges[uint]{}
// 	authoritySetChanges.append(1, 8)

// 	_, err := proveFinality[
// 		*BackendMock[string, uint, testHeader[string, uint],
// 			*BlockchainBackendMock[string, uint, testHeader[string, uint]]],
// 		string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 		testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]],
// 	](
// 		mockBackend,
// 		authoritySetChanges,
// 		6,
// 		true,
// 	)
// 	require.ErrorIs(t, err, errBlockNotInAuthoritySetChanges)
// }

// func TestFinalityProof_UsingAuthoritySetChangesWorks(t *testing.T) {
// 	ID := dummyAuthID(1)
// 	header7 := testHeader[string, uint]{
// 		NumberField: uint(7),
// 		HashField:   "hash7",
// 	}
// 	header8 := testHeader[string, uint]{
// 		NumberField:     uint(8),
// 		HashField:       "hash8",
// 		ParentHashField: "hash7",
// 	}

// 	dummyInfo := Info[uint]{
// 		FinalizedNumber: 8,
// 	}

// 	round := uint64(8)
// 	commit := createCommit(t, "hash8", uint(8), round, ID)
// 	grandpaJust := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Round:  round,
// 		Commit: commit,
// 	}

// 	encJust, err := scale.Marshal(grandpaJust)
// 	require.NoError(t, err)

// 	justifications := Justifications{Justification{
// 		EngineID:             GrandpaEngineID,
// 		EncodedJustification: encJust,
// 	}}

// 	mockBlockchain := NewBlockchainBackendMock[string, uint, testHeader[string, uint]](t)
// 	mockBlockchain.EXPECT().Info().Return(dummyInfo).Once()
// 	mockBlockchain.EXPECT().ExpectBlockHashFromID(uint(7)).Return("hash7", nil).Once()
// 	mockBlockchain.EXPECT().ExpectHeader("hash7").Return(header7, nil).Once()
// 	mockBlockchain.EXPECT().ExpectBlockHashFromID(uint(8)).Return("hash8", nil).Times(3)
// 	mockBlockchain.EXPECT().Justifications("hash8").Return(&justifications, nil).Times(1)
// 	mockBlockchain.EXPECT().ExpectHeader("hash8").Return(header8, nil).Once()

// 	mockBackend := NewBackendMock[string, uint, testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]]](t)
// 	mockBackend.EXPECT().Blockchain().Return(mockBlockchain).Times(8)

// 	// Authority set change at block 8, so the justification stored there will be used in the
// 	// FinalityProof for block 6
// 	authoritySetChanges := AuthoritySetChanges[uint]{}
// 	authoritySetChanges.append(0, 5)
// 	authoritySetChanges.append(1, 8)

// 	proofOf6, err := proveFinality[
// 		*BackendMock[string, uint, testHeader[string, uint],
// 			*BlockchainBackendMock[string, uint, testHeader[string, uint]]],
// 		string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 		testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]],
// 	](
// 		mockBackend,
// 		authoritySetChanges,
// 		6,
// 		true,
// 	)
// 	require.NoError(t, err)

// 	unknownHeaders := []testHeader[string, uint]{header7, header8}
// 	expFinalityProof := &FinalityProof[string, uint, testHeader[string, uint]]{
// 		Block:          "hash8",
// 		Justification:  encJust,
// 		UnknownHeaders: unknownHeaders,
// 	}
// 	require.Equal(t, expFinalityProof, proofOf6)

// 	mockBlockchain2 := NewBlockchainBackendMock[string, uint, testHeader[string, uint]](t)
// 	mockBlockchain2.EXPECT().Info().Return(dummyInfo).Once()
// 	mockBlockchain2.EXPECT().ExpectBlockHashFromID(uint(8)).Return("hash8", nil).Times(2)
// 	mockBlockchain2.EXPECT().Justifications("hash8").Return(&justifications, nil).Times(1)

// 	mockBackend2 := NewBackendMock[string, uint, testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]]](t)
// 	mockBackend2.EXPECT().Blockchain().Return(mockBlockchain2).Times(4)

// 	proofOf6WithoutUnknown, err := proveFinality[
// 		*BackendMock[string, uint, testHeader[string, uint],
// 			*BlockchainBackendMock[string, uint, testHeader[string, uint]]],
// 		string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 		testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]],
// 	](
// 		mockBackend2,
// 		authoritySetChanges,
// 		6,
// 		false,
// 	)
// 	require.NoError(t, err)

// 	expFinalityProof = &FinalityProof[string, uint, testHeader[string, uint]]{
// 		Block:         "hash8",
// 		Justification: encJust,
// 	}
// 	require.Equal(t, expFinalityProof, proofOf6WithoutUnknown)
// }

// func TestFinalityProof_InLastSetFailsWithoutLatest(t *testing.T) {
// 	dummyInfo := Info[uint]{
// 		FinalizedNumber: 8,
// 	}
// 	mockBlockchain := NewBlockchainBackendMock[string, uint, testHeader[string, uint]](t)
// 	mockBlockchain.EXPECT().Info().Return(dummyInfo).Once()

// 	mockBackend := NewBackendMock[string, uint, testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]]](t)
// 	mockBackend.EXPECT().Blockchain().Return(mockBlockchain).Times(1)
// 	mockBackend.EXPECT().Get(Key("grandpa_best_justification")).Return(nil, nil).Times(1)

// 	// No recent authority set change, so we are in the authoritySetChangeIDLatest set, and we will try to pickup
// 	// the best stored justification, for which there is none in this case.
// 	authoritySetChanges := AuthoritySetChanges[uint]{}
// 	authoritySetChanges.append(0, 5)

// 	proof, err := proveFinality[
// 		*BackendMock[string, uint, testHeader[string, uint],
// 			*BlockchainBackendMock[string, uint, testHeader[string, uint]]],
// 		string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 		testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]],
// 	](
// 		mockBackend,
// 		authoritySetChanges,
// 		6,
// 		true,
// 	)
// 	// When justification is not stored in db, return nil
// 	require.NoError(t, err)
// 	require.Nil(t, proof)
// }

// func TestFinalityProof_InLastSetUsingLatestJustificationWorks(t *testing.T) {
// 	ID := dummyAuthID(1)
// 	header7 := testHeader[string, uint]{
// 		NumberField: uint(7),
// 		HashField:   "hash7",
// 	}
// 	header8 := testHeader[string, uint]{
// 		NumberField:     uint(8),
// 		HashField:       "hash8",
// 		ParentHashField: "hash7",
// 	}

// 	dummyInfo := Info[uint]{
// 		FinalizedNumber: 8,
// 	}

// 	round := uint64(8)
// 	commit := createCommit(t, "hash8", uint(8), round, ID)
// 	grandpaJust := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Round:  round,
// 		Commit: commit,
// 	}

// 	encJust, err := scale.Marshal(grandpaJust)
// 	require.NoError(t, err)

// 	mockBlockchain := NewBlockchainBackendMock[string, uint, testHeader[string, uint]](t)
// 	mockBlockchain.EXPECT().Info().Return(dummyInfo).Once()
// 	mockBlockchain.EXPECT().ExpectBlockHashFromID(uint(7)).Return("hash7", nil).Once()
// 	mockBlockchain.EXPECT().ExpectHeader("hash7").Return(header7, nil).Once()
// 	mockBlockchain.EXPECT().ExpectBlockHashFromID(uint(8)).Return("hash8", nil).Times(2)
// 	mockBlockchain.EXPECT().ExpectHeader("hash8").Return(header8, nil).Once()

// 	mockBackend := NewBackendMock[string, uint, testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]]](t)
// 	mockBackend.EXPECT().Blockchain().Return(mockBlockchain).Times(6)
// 	mockBackend.EXPECT().Get(Key("grandpa_best_justification")).Return(&encJust, nil).Times(1)

// 	// No recent authority set change, so we are in the authoritySetChangeIDLatest set, and will pickup the best
// 	// stored justification (via mock get call)
// 	authoritySetChanges := AuthoritySetChanges[uint]{}
// 	authoritySetChanges.append(0, 5)

// 	proofOf6, err := proveFinality[
// 		*BackendMock[string, uint, testHeader[string, uint],
// 			*BlockchainBackendMock[string, uint, testHeader[string, uint]]],
// 		string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 		testHeader[string, uint],
// 		*BlockchainBackendMock[string, uint, testHeader[string, uint]],
// 	](
// 		mockBackend,
// 		authoritySetChanges,
// 		6,
// 		true,
// 	)
// 	require.NoError(t, err)

// 	unknownHeaders := []testHeader[string, uint]{header7, header8}

// 	expFinalityProof := &FinalityProof[string, uint, testHeader[string, uint]]{
// 		Block:          "hash8",
// 		Justification:  scale.MustMarshal(grandpaJust),
// 		UnknownHeaders: unknownHeaders,
// 	}
// 	require.Equal(t, expFinalityProof, proofOf6)
// }
