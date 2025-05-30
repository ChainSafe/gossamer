package gossipsupport

import (
	"context"
	"errors"
	"testing"
	"time"

	networkbridge "github.com/ChainSafe/gossamer/dot/parachain/network-bridge"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// --------  Mocks of the AuthorityDiscoveryService ------------- //
var _ networkbridge.AuthorityDiscoveryService = (*MockAuthorityDiscoveryServiceEmptyAddress)(nil)

type MockAuthorityDiscoveryServiceEmptyAddress struct{}

func (*MockAuthorityDiscoveryServiceEmptyAddress) GetPeerIDByAuthorityID(
	_ parachaintypes.AuthorityDiscoveryID,
) peer.ID {
	return "0"
}

func (*MockAuthorityDiscoveryServiceEmptyAddress) GetAuthorityIDsByPeerID(
	_ peer.ID,
) map[parachaintypes.AuthorityDiscoveryID]struct{} {
	return nil
}

func (*MockAuthorityDiscoveryServiceEmptyAddress) GetAddressesByAuthorityID(
	_ parachaintypes.AuthorityDiscoveryID,
) map[multiaddr.Multiaddr]struct{} {
	return nil
}

var _ networkbridge.AuthorityDiscoveryService = (*MockAuthorityDiscoveryServiceForIP4Address)(nil)

type MockAuthorityDiscoveryServiceForIP4Address struct{}

func (*MockAuthorityDiscoveryServiceForIP4Address) GetPeerIDByAuthorityID(
	_ parachaintypes.AuthorityDiscoveryID,
) peer.ID {
	return "0"
}

func (*MockAuthorityDiscoveryServiceForIP4Address) GetAuthorityIDsByPeerID(
	_ peer.ID,
) map[parachaintypes.AuthorityDiscoveryID]struct{} {
	return nil
}

func (*MockAuthorityDiscoveryServiceForIP4Address) GetAddressesByAuthorityID(
	_ parachaintypes.AuthorityDiscoveryID,
) map[multiaddr.Multiaddr]struct{} {
	addr, err := multiaddr.NewMultiaddrBytes([]byte{4, 1, 2, 3, 4, 6, 0, 80})
	if err != nil {
		panic(err)
	}
	return map[multiaddr.Multiaddr]struct{}{
		addr: {},
	}
}

var _ networkbridge.AuthorityDiscoveryService = (*MockAuthorityDiscoveryServiceForP2PAddress)(nil)

type MockAuthorityDiscoveryServiceForP2PAddress struct{}

func (*MockAuthorityDiscoveryServiceForP2PAddress) GetPeerIDByAuthorityID(
	_ parachaintypes.AuthorityDiscoveryID,
) peer.ID {
	return "0"
}

func (*MockAuthorityDiscoveryServiceForP2PAddress) GetAuthorityIDsByPeerID(
	_ peer.ID,
) map[parachaintypes.AuthorityDiscoveryID]struct{} {
	return nil
}

func (*MockAuthorityDiscoveryServiceForP2PAddress) GetAddressesByAuthorityID(
	_ parachaintypes.AuthorityDiscoveryID,
) map[multiaddr.Multiaddr]struct{} {
	addr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/4001/p2p/QmYwAPJzv5CZsnAzt8auVZRnX2pRhe84p2zjKBdTHZnr5k")
	if err != nil {
		panic(err)
	}
	return map[multiaddr.Multiaddr]struct{}{
		addr: {},
	}
}

// ------------- Tests start ------------- //
func TestIsSuperSet(t *testing.T) {
	superset := map[int]string{
		1: "a",
		2: "b",
		3: "c",
	}

	subset := map[int]string{
		1: "a",
	}
	assert.True(t, isSuperSet(superset, subset))

	superset2 := map[peer.ID]struct{}{
		peer.ID("1"): {},
		peer.ID("2"): {},
		peer.ID("3"): {},
	}

	subset2 := map[peer.ID]struct{}{
		peer.ID("1"): {},
		peer.ID("2"): {},
		peer.ID("3"): {},
	}
	assert.True(t, isSuperSet(superset2, subset2))

	superset3 := map[peer.ID]struct{}{
		peer.ID("1"): {},
	}

	subset3 := map[peer.ID]struct{}{
		peer.ID("1"): {},
		peer.ID("2"): {},
		peer.ID("3"): {},
	}
	assert.False(t, isSuperSet(superset3, subset3))
}

func TestEnsureIamAnAuthority(t *testing.T) {
	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	authorities := []parachaintypes.AuthorityDiscoveryID{
		{0x01},
	}

	keyIdx, err := ensureIamAnAuthority(testKs, authorities)
	assert.EqualError(t, err, "node is not a validator")
	assert.EqualValues(t, 0, keyIdx)

	authorities = []parachaintypes.AuthorityDiscoveryID{
		parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
	}

	keyIdx, err = ensureIamAnAuthority(testKs, authorities)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, keyIdx)
}

func TestCheckConnectivity(t *testing.T) {
	gs := NewGossipSupport(nil, nil, nil)
	gs.connectedAuthorities = map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID{
		{0x01}: "01",
	}

	addr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	assert.Nil(t, err)

	gs.resolvedAuthorities = map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}{
		{0x02}: {addr: {}},
		{0x03}: {addr: {}},
	}

	gs.checkConnectivity()
}

func TestGetKeyIndexAndUpdateMetricsNoLongerAuthority(t *testing.T) {
	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	gs := NewGossipSupport(testKs, nil, nil)

	authorities := []parachaintypes.AuthorityDiscoveryID{
		{0x01},
	}

	sessionInfo := &parachaintypes.SessionInfo{DiscoveryKeys: authorities}
	keyIdx, err := gs.getKeyIndexAndUpdateMetrics(sessionInfo)

	assert.EqualError(t, err, "node is not a validator")
	assert.EqualValues(t, 0, keyIdx)
}

func TestGetKeyIndexAndUpdateMetricsIsAuthorityNow(t *testing.T) {
	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	bobKeypair := keyring.Bob().(*sr25519.Keypair)
	charlieKeypair := keyring.Charlie().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	gs := NewGossipSupport(testKs, nil, nil)

	authorities := []parachaintypes.AuthorityDiscoveryID{
		parachaintypes.AuthorityDiscoveryID(bobKeypair.Public().Encode()),
		parachaintypes.AuthorityDiscoveryID(charlieKeypair.Public().Encode()),
		parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
	}

	// The subset of authorities participating in parachain consensus is greater
	// than the key index of the current authority
	sessionInfo := &parachaintypes.SessionInfo{
		DiscoveryKeys: authorities,
		Validators:    []parachaintypes.ValidatorID{{0x01}, {0x02}, {0x03}},
	}
	keyIdx, err := gs.getKeyIndexAndUpdateMetrics(sessionInfo)

	assert.Nil(t, err)
	assert.EqualValues(t, 2, keyIdx)

	// The subset of authorities participating in parachain consensus is less
	// than the key index of the current authority
	sessionInfo = &parachaintypes.SessionInfo{DiscoveryKeys: authorities, Validators: []parachaintypes.ValidatorID{{}}}
	keyIdx, err = gs.getKeyIndexAndUpdateMetrics(sessionInfo)

	assert.Nil(t, err)
	assert.EqualValues(t, 2, keyIdx)
}

func TestAuthoritiesPastPresentFutureGetAuthorityError(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	runtimeMock.EXPECT().GrandpaAuthorities().Return(nil, errors.New("something is off")).Times(1)
	future, err := authoritiesPastPresentFuture(runtimeMock)
	assert.Nil(t, future)
	assert.EqualError(t, err, "something is off")
}

func TestAuthoritiesPastPresentFutureGetAuthorityOk(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	auth := types.NewAuthority(aliceKeypair.Public(), 0)
	runtimeMock.EXPECT().GrandpaAuthorities().Return([]types.Authority{*auth}, nil).Times(1)
	future, err := authoritiesPastPresentFuture(runtimeMock)

	assert.Nil(t, err)
	assert.EqualValues(t, []parachaintypes.AuthorityDiscoveryID{
		parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
	}, future)
}

func TestRemoveAllControlled(t *testing.T) {
	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	bobKeypair := keyring.Bob().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	authorities := []parachaintypes.AuthorityDiscoveryID{
		parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
		parachaintypes.AuthorityDiscoveryID(bobKeypair.Public().Encode()),
	}

	re, counter := removeAllControlled(testKs, authorities)
	assert.EqualValues(t, []parachaintypes.AuthorityDiscoveryID{
		parachaintypes.AuthorityDiscoveryID(bobKeypair.Public().Encode()),
	}, re)
	assert.EqualValues(t, 1, counter)
}

func TestResolveAuthorities(t *testing.T) {
	gs := NewGossipSupport(nil, nil, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceForIP4Address{}

	validatorAddrs, resolved, failure := gs.resolveAuthorities([]parachaintypes.AuthorityDiscoveryID{
		{0x01},
	})

	assert.EqualValues(t, 1, len(validatorAddrs))
	assert.EqualValues(t, 1, len(resolved))
	assert.Zero(t, failure)
}

func TestIssueConnectionRequestToChangedOldAddressesNotNilIP4Address(t *testing.T) {
	overseerChan := make(chan any)

	gs := NewGossipSupport(nil, overseerChan, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceForIP4Address{}

	addr, err := multiaddr.NewMultiaddrBytes([]byte{4, 1, 2, 3, 4, 6, 0, 80})
	assert.Nil(t, err)

	gs.resolvedAuthorities = map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}{
		{0x01}: {addr: {}},
	}

	done := make(chan struct{})
	go func() {
		for {
			msg := <-overseerChan
			if _, ok := msg.(networkbridgemessages.AddToResolvedValidators); !ok {
				t.Error("receiving the wrong type of msg")
				return
			}
			done <- struct{}{}
		}
	}()

	gs.issueConnectionRequestToChanged([]parachaintypes.AuthorityDiscoveryID{
		{0x01},
	})

	select {
	case <-done:
		t.Log("test completed successfully")
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestIssueConnectionRequestToChangedOldAddressesNotNil(t *testing.T) {
	overseerChan := make(chan any)

	gs := NewGossipSupport(nil, overseerChan, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceForP2PAddress{}

	addr, err := multiaddr.NewMultiaddrBytes([]byte{4, 1, 2, 3, 4, 6, 0, 80})
	assert.Nil(t, err)

	gs.resolvedAuthorities = map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}{
		{0x01}: {addr: {}},
	}

	gs.issueConnectionRequestToChanged([]parachaintypes.AuthorityDiscoveryID{
		{0x01},
	})

	assert.EqualValues(t, map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}{
		{0x01}: {addr: {}},
	},
		gs.resolvedAuthorities)
}

func TestIssueConnectionRequestToChangedOldAddressesIsNil(t *testing.T) {
	overseerChan := make(chan any)

	gs := NewGossipSupport(nil, overseerChan, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceForP2PAddress{}

	gs.issueConnectionRequestToChanged([]parachaintypes.AuthorityDiscoveryID{
		{0x01},
	})

	assert.EqualValues(t, map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}{},
		gs.resolvedAuthorities)
}

func TestIssueConnectionRequest(t *testing.T) {
	overseerChan := make(chan any)

	gs := NewGossipSupport(nil, overseerChan, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceForP2PAddress{}

	done := make(chan struct{})
	go func() {
		for {
			msg := <-overseerChan
			if _, ok := msg.(networkbridgemessages.ConnectTOResolvedValidators); !ok {
				t.Error("receiving the wrong type of msg")
				return
			}
			done <- struct{}{}
		}
	}()

	gs.issueConnectionRequest([]parachaintypes.AuthorityDiscoveryID{
		{0x01},
	})

	select {
	case <-done:
		t.Log("test completed successfully")
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestIssueConnectionRequestIssueAnotherRequestForTheSameSession(t *testing.T) {
	overseerChan := make(chan any)

	gs := NewGossipSupport(nil, overseerChan, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceEmptyAddress{}

	done := make(chan struct{})
	go func() {
		for {
			msg := <-overseerChan
			if _, ok := msg.(networkbridgemessages.ConnectTOResolvedValidators); !ok {
				t.Error("receiving the wrong type of msg")
				return
			}
			done <- struct{}{}
		}
	}()

	gs.issueConnectionRequest([]parachaintypes.AuthorityDiscoveryID{
		{0x01},
	})

	select {
	case <-done:
		t.Log("test completed successfully")
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestIssueConnectionRequestIssueAnotherRequestForTheSameSessionFailureNotNil(t *testing.T) {
	overseerChan := make(chan any)

	gs := NewGossipSupport(nil, overseerChan, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceEmptyAddress{}
	ts := time.Now().Add(11 * time.Minute)
	gs.failureStart = &ts

	done := make(chan struct{})
	go func() {
		for {
			msg := <-overseerChan
			if _, ok := msg.(networkbridgemessages.ConnectTOResolvedValidators); !ok {
				t.Error("receiving the wrong type of msg")
				return
			}
			done <- struct{}{}
		}
	}()

	gs.issueConnectionRequest([]parachaintypes.AuthorityDiscoveryID{
		{0x01},
	})

	select {
	case <-done:
		t.Log("test completed successfully")
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestBuildTopologyForLastFinalizedIfNeeded(t *testing.T) {
	gs := NewGossipSupport(nil, nil, nil)
	gs.minKnownSession = parachaintypes.SessionIndex(10)
	finalizedNeededSession := parachaintypes.SessionIndex(1)
	gs.finalizedNeededSession = &finalizedNeededSession
	currentSessionIdx := parachaintypes.SessionIndex(5)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	gs.keystore = testKs

	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	runtimeMock.EXPECT().FinalizeBlock().Return(&types.Header{Number: 1}, nil).Times(1)

	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(2), nil).Times(1)

	mockedSessionInfo := &parachaintypes.SessionInfo{
		DiscoveryKeys: []parachaintypes.AuthorityDiscoveryID{
			parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
		},
	}
	runtimeMock.EXPECT().ParachainHostSessionInfo(parachaintypes.SessionIndex(2)).Return(mockedSessionInfo, nil).Times(1)

	err = gs.buildTopologyForLastFinalizedIfNeeded(currentSessionIdx, runtimeMock)
	assert.Nil(t, err)

	assert.EqualValues(t, parachaintypes.SessionIndex(2), *gs.finalizedNeededSession)
}

func TestProcessBlockFinalizedSignalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := NewMockBlockState(ctrl)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{0x01}).Return(
		nil, errors.New("something is off")).Times(1)

	signal := parachaintypes.BlockFinalizedSignal{
		Hash: common.Hash{0x01},
	}

	gs := NewGossipSupport(nil, nil, blockAPIMock)
	err := gs.ProcessBlockFinalizedSignal(signal)

	assert.EqualError(t, err, "something is off")
}

func TestProcessBlockFinalizedSignal(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := NewMockBlockState(ctrl)
	runtimeMock := NewMockInstance(ctrl)

	gs := NewGossipSupport(nil, nil, blockAPIMock)
	lastSessionInd := parachaintypes.SessionIndex(5)
	gs.lastSessionIndex = &lastSessionInd
	gs.minKnownSession = parachaintypes.SessionIndex(10)
	finalizedNeededSession := parachaintypes.SessionIndex(1)
	gs.finalizedNeededSession = &finalizedNeededSession

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)
	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)
	gs.keystore = testKs

	blockAPIMock.EXPECT().GetRuntime(common.Hash{0x01}).Return(runtimeMock, nil).Times(1)
	runtimeMock.EXPECT().FinalizeBlock().Return(nil, errors.New("something is off")).Times(1)
	signal := parachaintypes.BlockFinalizedSignal{
		Hash: common.Hash{0x01},
	}

	err = gs.ProcessBlockFinalizedSignal(signal)
	assert.EqualError(t, err, "something is off")
}

func TestUpdateAuthorityIDsAuthorityIDsEmpty(t *testing.T) {
	overseerChan := make(chan any)
	gs := NewGossipSupport(nil, overseerChan, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceForP2PAddress{}

	var peerID parachaintypes.PeerID
	addr := gs.authorityDiscovery.GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{})
	for a := range addr {
		_, p := peer.SplitAddr(a)
		peerID = parachaintypes.PeerID(p)
	}
	gs.connectedPeers = map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{
		peerID: {{0x01}: {}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case msg := <-overseerChan:
				if _, ok := msg.(networkbridgemessages.UpdateAuthorityIDs); !ok {
					t.Error("receiving wrong msg type")
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	var authorities []parachaintypes.AuthorityDiscoveryID
	gs.updateAuthorityIDs(authorities)
	assert.EqualValues(t, map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID{}, gs.connectedAuthorities)
	assert.EqualValues(t,
		map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{peerID: {{0x01}: {}}},
		gs.connectedPeers)

	cancel()
}

func TestUpdateAuthorityIDsConnectedPeersGotAllRemoved(t *testing.T) {
	overseerChan := make(chan any)
	gs := NewGossipSupport(nil, overseerChan, nil)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceForP2PAddress{}

	var peerID parachaintypes.PeerID
	addr := gs.authorityDiscovery.GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{})
	for a := range addr {
		_, p := peer.SplitAddr(a)
		peerID = parachaintypes.PeerID(p)
	}
	gs.connectedPeers = map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{
		peerID: {{0x01}: {}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case msg := <-overseerChan:
				if _, ok := msg.(networkbridgemessages.UpdateAuthorityIDs); !ok {
					t.Error("receiving wrong msg type")
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	authorities := []parachaintypes.AuthorityDiscoveryID{
		{0x01},
	}
	gs.updateAuthorityIDs(authorities)
	assert.EqualValues(t,
		map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID{{0x01}: peerID},
		gs.connectedAuthorities)
	assert.EqualValues(t,
		map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{peerID: {{0x01}: {}}},
		gs.connectedPeers)

	cancel()
}

func TestProcessActiveLeavesUpdateSignal(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := NewMockBlockState(ctrl)
	runtimeMock := NewMockInstance(ctrl)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{0x01}).Return(runtimeMock, nil).Times(1)
	auth := types.NewAuthority(aliceKeypair.Public(), 0)
	runtimeMock.EXPECT().GrandpaAuthorities().Return([]types.Authority{*auth}, nil).Times(1)
	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(2), nil).Times(2)
	runtimeMock.EXPECT().FinalizeBlock().Return(&types.Header{Number: 1}, nil).Times(1)

	mockedSessionInfo := &parachaintypes.SessionInfo{
		DiscoveryKeys: []parachaintypes.AuthorityDiscoveryID{
			parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
		},
	}
	runtimeMock.EXPECT().ParachainHostSessionInfo(parachaintypes.SessionIndex(2)).Return(mockedSessionInfo, nil).Times(1)

	overseerChan := make(chan any)
	gs := NewGossipSupport(testKs, overseerChan, blockAPIMock)
	gs.authorityDiscovery = &MockAuthorityDiscoveryServiceForP2PAddress{}
	lastFailure := time.Now().Add(11 * time.Minute)
	lastConnectionRequest := time.Now().Add(20 * time.Minute)
	gs.lastFailure = &lastFailure
	gs.lastConnectionRequest = &lastConnectionRequest

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case msg := <-overseerChan:
				switch msg.(type) {
				case networkbridgemessages.UpdateAuthorityIDs, networkbridgemessages.ConnectTOResolvedValidators,
					networkbridgemessages.AddToResolvedValidators:
					continue
				default:
					t.Error("receiving wrong msg type")
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	signal := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash: common.Hash{0x01},
		},
	}
	err = gs.ProcessActiveLeavesUpdateSignal(signal)
	assert.Nil(t, err)

	assert.EqualValues(t, parachaintypes.SessionIndex(2), *gs.lastSessionIndex)

	cancel()
}
