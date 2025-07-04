package gossipsupport

import (
	"context"
	"errors"
	"testing"
	"time"

	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
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

func TestIsSuperSet(t *testing.T) {
	superset := map[peer.ID]struct{}{
		peer.ID("1"): {},
		peer.ID("2"): {},
		peer.ID("3"): {},
	}

	subset := map[peer.ID]struct{}{
		peer.ID("1"): {},
		peer.ID("2"): {},
		peer.ID("3"): {},
	}
	assert.True(t, isSuperSet(superset, subset))

	superset2 := map[peer.ID]struct{}{
		peer.ID("1"): {},
	}

	subset2 := map[peer.ID]struct{}{
		peer.ID("1"): {},
		peer.ID("2"): {},
		peer.ID("3"): {},
	}
	assert.False(t, isSuperSet(superset2, subset2))
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

	keyIdx, ok := ensureIamAnAuthority(testKs, authorities)
	assert.False(t, ok)
	assert.EqualValues(t, 0, keyIdx)

	authorities = []parachaintypes.AuthorityDiscoveryID{
		parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
	}

	keyIdx, ok = ensureIamAnAuthority(testKs, authorities)
	assert.True(t, ok)
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

func TestGetKeyIndexAndUpdateMetrics(t *testing.T) {
	t.Parallel()

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	gs := NewGossipSupport(testKs, nil, nil)

	t.Run("no_longer_authority", func(t *testing.T) {
		t.Parallel()

		authorities := []parachaintypes.AuthorityDiscoveryID{
			{0x01},
		}

		sessionInfo := &parachaintypes.SessionInfo{DiscoveryKeys: authorities}
		keyIdx, err := gs.getKeyIndexAndUpdateMetrics(sessionInfo)

		assert.EqualError(t, err, "node is not a validator")
		assert.EqualValues(t, 0, keyIdx)
	})

	t.Run("is_authority_now", func(t *testing.T) {
		t.Parallel()

		bobKeypair := keyring.Bob().(*sr25519.Keypair)
		charlieKeypair := keyring.Charlie().(*sr25519.Keypair)

		authorities := []parachaintypes.AuthorityDiscoveryID{
			parachaintypes.AuthorityDiscoveryID(bobKeypair.Public().Encode()),
			parachaintypes.AuthorityDiscoveryID(charlieKeypair.Public().Encode()),
			parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
		}

		// The subset of authorities participating in parachain consensus is greater
		// than the key index of the current authority
		sessionInfo := &parachaintypes.SessionInfo{
			DiscoveryKeys: authorities,
			Validators:    []parachaintypes.ValidatorPublicKey{{0x01}, {0x02}, {0x03}},
		}
		keyIdx, err := gs.getKeyIndexAndUpdateMetrics(sessionInfo)

		assert.Nil(t, err)
		assert.EqualValues(t, 2, keyIdx)

		// The subset of authorities participating in parachain consensus is less
		// than the key index of the current authority
		sessionInfo = &parachaintypes.SessionInfo{
			DiscoveryKeys: authorities,
			Validators: []parachaintypes.ValidatorPublicKey{
				{},
			},
		}
		keyIdx, err = gs.getKeyIndexAndUpdateMetrics(sessionInfo)

		assert.Nil(t, err)
		assert.EqualValues(t, 2, keyIdx)
	})
}

func TestAuthoritiesPastPresentFuture(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	t.Run("get_authorities_error", func(t *testing.T) {
		t.Parallel()

		runtimeMock.EXPECT().GrandpaAuthorities().Return(nil, errors.New("something is off")).Times(1)
		future, err := authoritiesPastPresentFuture(runtimeMock)
		assert.Nil(t, future)
		assert.EqualError(t, err, "something is off")
	})

	t.Run("get_authorities_ok", func(t *testing.T) {
		t.Parallel()

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
	})
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
	ctrl := gomock.NewController(t)
	adsMock := NewMockAuthorityDiscoveryService(ctrl)

	gs := NewGossipSupport(nil, nil, nil)
	gs.authorityDiscovery = adsMock

	addr, err := multiaddr.NewMultiaddrBytes([]byte{4, 1, 2, 3, 4, 6, 0, 80})
	assert.Nil(t, err)

	authIDs := []parachaintypes.AuthorityDiscoveryID{
		{0x01},
	}

	adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{0x01}).
		Return(map[multiaddr.Multiaddr]struct{}{addr: {}}).Times(1)

	validatorAddrs, resolved, failure := gs.resolveAuthorities(authIDs)

	assert.EqualValues(t, 1, len(validatorAddrs))
	assert.EqualValues(t, 1, len(resolved))
	assert.Zero(t, failure)
}

func TestIssueConnectionRequestToChangedOldAddresses(t *testing.T) {
	ctrl := gomock.NewController(t)
	adsMock := NewMockAuthorityDiscoveryService(ctrl)

	t.Run("changed_address_not_nil_ip4_address", func(t *testing.T) {
		overseerChan := make(chan any)

		gs := NewGossipSupport(nil, overseerChan, nil)
		gs.authorityDiscovery = adsMock

		addr, err := multiaddr.NewMultiaddrBytes([]byte{4, 1, 2, 3, 4, 6, 0, 80})
		assert.Nil(t, err)

		adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{0x01}).
			Return(map[multiaddr.Multiaddr]struct{}{addr: {}}).Times(1)

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
	})

	t.Run("changed_address_not_nil_p2p_address", func(t *testing.T) {
		overseerChan := make(chan any)

		gs := NewGossipSupport(nil, overseerChan, nil)

		gs.authorityDiscovery = adsMock
		p2pAddr, err := multiaddr.NewMultiaddr("" +
			"/ip4/127.0.0.1/tcp/4001/p2p/QmYwAPJzv5CZsnAzt8auVZRnX2pRhe84p2zjKBdTHZnr5k")
		assert.Nil(t, err)

		adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{0x01}).
			Return(map[multiaddr.Multiaddr]struct{}{p2pAddr: {}}).Times(1)

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
	})

	t.Run("changed_address_is_nil", func(t *testing.T) {
		overseerChan := make(chan any)

		gs := NewGossipSupport(nil, overseerChan, nil)
		gs.authorityDiscovery = adsMock
		p2pAddr, err := multiaddr.NewMultiaddr("" +
			"/ip4/127.0.0.1/tcp/4001/p2p/QmYwAPJzv5CZsnAzt8auVZRnX2pRhe84p2zjKBdTHZnr5k")
		assert.Nil(t, err)

		adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{0x01}).
			Return(map[multiaddr.Multiaddr]struct{}{p2pAddr: {}}).Times(1)

		gs.issueConnectionRequestToChanged([]parachaintypes.AuthorityDiscoveryID{
			{0x01},
		})

		assert.EqualValues(t, map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}{},
			gs.resolvedAuthorities)
	})
}

func TestIssueConnectionRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	adsMock := NewMockAuthorityDiscoveryService(ctrl)

	t.Run("simple_connection_request", func(t *testing.T) {
		overseerChan := make(chan any)

		gs := NewGossipSupport(nil, overseerChan, nil)
		gs.authorityDiscovery = adsMock
		p2pAddr, err := multiaddr.NewMultiaddr("" +
			"/ip4/127.0.0.1/tcp/4001/p2p/QmYwAPJzv5CZsnAzt8auVZRnX2pRhe84p2zjKBdTHZnr5k")
		assert.Nil(t, err)

		adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{0x01}).
			Return(map[multiaddr.Multiaddr]struct{}{p2pAddr: {}}).Times(1)

		done := make(chan struct{})
		go func() {
			for {
				msg := <-overseerChan
				if _, ok := msg.(networkbridgemessages.ConnectToResolvedValidators); !ok {
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
	})

	t.Run("issue_another_request_for_the_same_session", func(t *testing.T) {
		overseerChan := make(chan any)

		gs := NewGossipSupport(nil, overseerChan, nil)
		gs.authorityDiscovery = adsMock

		adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{0x01}).Return(nil).Times(1)

		done := make(chan struct{})
		go func() {
			for {
				msg := <-overseerChan
				if _, ok := msg.(networkbridgemessages.ConnectToResolvedValidators); !ok {
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
	})

	t.Run("issue_another_request_for_the_same_session_failure_for_nil", func(t *testing.T) {
		overseerChan := make(chan any)

		gs := NewGossipSupport(nil, overseerChan, nil)
		gs.authorityDiscovery = adsMock

		adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{0x01}).Return(nil).Times(1)

		ts := time.Now().Add(11 * time.Minute)
		gs.failureStart = &ts

		done := make(chan struct{})
		go func() {
			for {
				msg := <-overseerChan
				if _, ok := msg.(networkbridgemessages.ConnectToResolvedValidators); !ok {
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
	})
}

func TestBuildTopologyForLastFinalizedIfNeeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)
	blockAPIMock := NewMockBlockState(ctrl)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	overseerChan := make(chan any)
	gs := NewGossipSupport(testKs, overseerChan, blockAPIMock)

	// this is to unblock the overseerChan message called inside updateGossipTopology,
	// but we don't care the msg details since our goal is testing buildTopologyForLastFinalizedIfNeeded method
	go func() {
		<-overseerChan
	}()

	gs.minKnownSession = parachaintypes.SessionIndex(10)
	finalizedNeededSession := parachaintypes.SessionIndex(1)
	gs.finalizedNeededSession = &finalizedNeededSession
	currentSessionIdx := parachaintypes.SessionIndex(5)

	blockAPIMock.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{Number: 1}, nil).Times(1)

	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(2), nil).Times(1)

	mockedSessionInfo := &parachaintypes.SessionInfo{
		DiscoveryKeys: []parachaintypes.AuthorityDiscoveryID{
			parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
		},
	}
	runtimeMock.EXPECT().ParachainHostSessionInfo(parachaintypes.SessionIndex(2)).Return(mockedSessionInfo, nil).
		Times(1)

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
	blockAPIMock.EXPECT().GetHighestFinalisedHeader().Return(nil, errors.New("something is off")).Times(1)
	signal := parachaintypes.BlockFinalizedSignal{
		Hash: common.Hash{0x01},
	}

	err = gs.ProcessBlockFinalizedSignal(signal)
	assert.EqualError(t, err, "something is off")
}

func TestUpdateAuthorityIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	adsMock := NewMockAuthorityDiscoveryService(ctrl)

	t.Run("authorityIDs_empty", func(t *testing.T) {
		overseerChan := make(chan any)

		gs := NewGossipSupport(nil, overseerChan, nil)
		gs.authorityDiscovery = adsMock

		p2pAddr, err := multiaddr.NewMultiaddr("" +
			"/ip4/127.0.0.1/tcp/4001/p2p/QmYwAPJzv5CZsnAzt8auVZRnX2pRhe84p2zjKBdTHZnr5k")
		assert.Nil(t, err)

		var peerID parachaintypes.PeerID
		_, p := peer.SplitAddr(p2pAddr)
		peerID = parachaintypes.PeerID(p)

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
	})

	t.Run("connected_peers_got_all_removed", func(t *testing.T) {
		overseerChan := make(chan any)
		gs := NewGossipSupport(nil, overseerChan, nil)

		gs.authorityDiscovery = adsMock
		p2pAddr, err := multiaddr.NewMultiaddr("" +
			"/ip4/127.0.0.1/tcp/4001/p2p/QmYwAPJzv5CZsnAzt8auVZRnX2pRhe84p2zjKBdTHZnr5k")
		assert.Nil(t, err)

		adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID{0x01}).
			Return(map[multiaddr.Multiaddr]struct{}{p2pAddr: {}}).Times(1)

		var peerID parachaintypes.PeerID
		_, p := peer.SplitAddr(p2pAddr)
		peerID = parachaintypes.PeerID(p)

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
	})
}

func TestProcessActiveLeavesUpdateSignal(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := NewMockBlockState(ctrl)
	runtimeMock := NewMockInstance(ctrl)
	adsMock := NewMockAuthorityDiscoveryService(ctrl)

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
	blockAPIMock.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{Number: 1}, nil).Times(1)

	mockedSessionInfo := &parachaintypes.SessionInfo{
		DiscoveryKeys: []parachaintypes.AuthorityDiscoveryID{
			parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode()),
		},
	}
	runtimeMock.EXPECT().ParachainHostSessionInfo(parachaintypes.SessionIndex(2)).Return(mockedSessionInfo, nil).
		Times(1)

	overseerChan := make(chan any)
	gs := NewGossipSupport(testKs, overseerChan, blockAPIMock)

	gs.authorityDiscovery = adsMock
	p2pAddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/4001/p2p/QmYwAPJzv5CZsnAzt8auVZRnX2pRhe84p2zjKBdTHZnr5k")
	assert.Nil(t, err)

	adsMock.EXPECT().GetAddressesByAuthorityID(parachaintypes.AuthorityDiscoveryID(aliceKeypair.Public().Encode())).
		Return(map[multiaddr.Multiaddr]struct{}{p2pAddr: {}}).Times(1)

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
				case networkbridgemessages.UpdateAuthorityIDs, networkbridgemessages.ConnectToResolvedValidators,
					networkbridgemessages.AddToResolvedValidators, networkbridgemessages.NewGossipTopology:
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

func TestProcessPeerConnectedEvent(t *testing.T) {
	gs := NewGossipSupport(nil, nil, nil)

	eventWithEmptyAuth := networkbridgeevents.PeerConnected{
		PeerID:                "1",
		AuthorityDiscoveryIDs: nil,
	}

	assert.EqualValues(t, gs.connectedPeers,
		map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{})

	gs.processPeerConnectedEvent(eventWithEmptyAuth)

	assert.EqualValues(t, gs.connectedPeers, map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{
		"1": {},
	})

	eventWithAuth := networkbridgeevents.PeerConnected{
		PeerID:                "2",
		AuthorityDiscoveryIDs: &[]parachaintypes.AuthorityDiscoveryID{{0x02}},
	}

	assert.EqualValues(t, gs.connectedPeers, map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{
		"1": {},
	})

	gs.processPeerConnectedEvent(eventWithAuth)

	assert.EqualValues(t, gs.connectedAuthorities, map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID{
		{0x02}: "2",
	})
	assert.EqualValues(t, gs.connectedPeers, map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{
		"1": {},
		"2": {{0x02}: {}},
	})
}

func TestProcessPeerDisconnectedEvent(t *testing.T) {
	gs := NewGossipSupport(nil, nil, nil)
	gs.connectedPeers["1"] = map[parachaintypes.AuthorityDiscoveryID]struct{}{{0x01}: {}}
	gs.connectedAuthorities[parachaintypes.AuthorityDiscoveryID{0x01}] = "1"

	event := networkbridgeevents.PeerDisconnected{
		PeerID: "1",
	}

	gs.processPeerDisconnectedEvent(event)
	assert.EqualValues(t, gs.connectedPeers,
		map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}{})
	assert.EqualValues(t, gs.connectedAuthorities, map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID{})
}

func TestUpdateGossipTopology(t *testing.T) {
	overseerChan := make(chan any)
	gs := NewGossipSupport(nil, overseerChan, nil)
	ourIndex := uint(1)
	sessionIndex := parachaintypes.SessionIndex(2)
	authorities := []parachaintypes.AuthorityDiscoveryID{
		{0x01}, {0x02}, {0x03},
	}

	done := make(chan struct{})
	go func() {
		msg := <-overseerChan
		val, ok := msg.(networkbridgemessages.NewGossipTopology)
		if !ok {
			t.Error("Received wrong msg type")
			return
		} else {
			assert.EqualValues(t, parachaintypes.ValidatorIndex(ourIndex), *val.LocalIndex)
			assert.EqualValues(t, sessionIndex, val.Session)
			assert.EqualValues(t, 3, len(val.CanonicalShuffling))
			assert.EqualValues(t, 3, len(val.ShuffledIndices))

			// Check that all AuthorityDiscoveryID are present
			authIDMap := make(map[parachaintypes.AuthorityDiscoveryID]bool)
			for _, pair := range val.CanonicalShuffling {
				authIDMap[pair.AuthorityDiscoveryID] = true
			}
			for _, authID := range authorities {
				assert.True(t, authIDMap[authID], "missing AuthorityDiscoveryID %s", authID)
			}
		}
		done <- struct{}{}

	}()

	err := gs.updateGossipTopology(ourIndex, authorities, sessionIndex)
	assert.Nil(t, err)

	select {
	case <-done:
		t.Log("test completed successfully")
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out")
	}
}
