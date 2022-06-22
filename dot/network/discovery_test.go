// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	badger "github.com/ipfs/go-ds-badger2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDiscovery(t *testing.T, num int) []*discovery {
	t.Helper()

	var discs []*discovery
	for i := 0; i < num; i++ {
		config := &Config{
			BasePath:    t.TempDir(),
			Port:        availablePort(t),
			NoBootstrap: true,
			NoMDNS:      true,
		}

		srvc := createTestService(t, config)

		opts := badger.DefaultOptions
		opts.InMemory = true

		ds, err := badger.NewDatastore("", &opts)
		require.NoError(t, err)
		disc := &discovery{
			ctx: srvc.ctx,
			h:   srvc.host.p2pHost,
			ds:  ds,
		}

		go disc.start()
		discs = append(discs, disc)
	}
	return discs
}

func connectNoSync(ctx context.Context, t *testing.T, a, b *discovery) {
	t.Helper()

	idB := b.h.ID()
	addrB := b.h.Peerstore().Addrs(idB)
	require.NotEqual(t, 0, len(addrB), "peers setup incorrectly: no local address")

	a.h.Peerstore().AddAddrs(idB, addrB, time.Minute)
	pi := peer.AddrInfo{ID: idB}

	err := a.h.Connect(ctx, pi)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = a.h.Connect(ctx, pi)
	}

	require.NoError(t, err)
}

// Set `NoMDNS` to true and test routing via kademlia DHT service.
func TestKadDHT(t *testing.T) {
	if testing.Short() {
		return
	}

	t.Parallel()

	// setup 3 nodes
	nodes := newTestDiscovery(t, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// connects node 0 and node 2
	connectNoSync(ctx, t, nodes[2], nodes[0])

	time.Sleep(startDHTTimeout + 1)

	// node 2 doesnt know about node 1 then should return error
	_, err := nodes[2].dht.FindPeer(ctx, nodes[1].h.ID())
	require.ErrorIs(t, err, routing.ErrNotFound)

	// connects node 1 and node 0
	connectNoSync(ctx, t, nodes[1], nodes[0])

	time.Sleep(startDHTTimeout + 1)

	// node 2 should know node 1 because both are connected to 0
	_, err = nodes[2].dht.FindPeer(ctx, nodes[1].h.ID())
	require.NoError(t, err)
}

func TestBeginDiscovery(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	addrInfoB := nodeB.host.addrInfo()
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	err = nodeA.host.discovery.start()
	require.NoError(t, err)

	err = nodeB.host.discovery.start()
	require.NoError(t, err)
}

func connectNodes(t *testing.T, nodesService []*Service) {
	if len(nodesService) <= 1 {
		return
	}

	// wait a amount of time to fully start the nodes
	time.Sleep(3 * time.Second)

	pivotNode := nodesService[0]
	pivotNodeAddr := pivotNode.host.addrInfo()

	var wg sync.WaitGroup
	const maxAttempts = 5

	errors := make([]chan error, len(nodesService)-1)

	for idx := 1; idx < len(nodesService); idx++ {
		nodeAt := nodesService[idx]
		errors[idx-1] = make(chan error)

		wg.Add(1)
		go func(service *Service, wg *sync.WaitGroup, errCh chan<- error) {
			defer wg.Done()
			defer close(errCh)

			for att := 0; att < maxAttempts; att++ {
				err := service.host.connect(pivotNodeAddr)
				if err == nil {
					fmt.Printf("connection ok %s - %s\n", service.host.addrInfo(), pivotNodeAddr)
					return
				}

				fmt.Printf("error %s\n", err)
				if failedToDial(err) {
					time.Sleep(TestBackoffTimeout)
				} else {
					errCh <- fmt.Errorf("problems while connecting: %w", err)
					return
				}
			}

			errCh <- fmt.Errorf("cannot establish connection between %s - %s", pivotNodeAddr, service.host.addrInfo())
		}(nodeAt, &wg, errors[idx-1])
	}

	for _, errCh := range errors {
		for err := range errCh {
			assert.NoError(t, err)
		}
	}

	wg.Wait()
}

func TestBeginDiscovery_ThreeNodes(t *testing.T) {
	t.Parallel()
	const amount = 3

	nodesService := make([]*Service, 0, amount)
	for node := amount; node > 0; node-- {
		nodeConfig := &Config{
			BasePath:    t.TempDir(),
			Port:        availablePort(t),
			NoBootstrap: true,
			NoMDNS:      true,
		}

		nodeService := createTestService(t, nodeConfig)
		nodeService.noGossip = true

		nodesService = append(nodesService, nodeService)
	}

	// servASub, err := nodesService[0].host.p2pHost.EventBus().Subscribe(event.WildcardSubscription)
	// require.NoError(t, err)

	var wg sync.WaitGroup

	// wg.Add(1)

	// go func() {
	// 	defer wg.Done()

	// 	for d := range servASub.Out() {
	// 		fmt.Printf("(%T) %v\n", d, d)
	// 	}
	// }()

	// A -> B
	// A -> C

	// A (2)
	// B (1)
	// C (1)

	connectNodes(t, nodesService)

	fmt.Println(">>>>>")

	fmt.Println(nodesService[0].host.p2pHost.Network().Conns())

	wg.Add(1)
	go func() {
		defer wg.Done()
		// begin advertising and discovery for all nodes
		err := nodesService[0].host.discovery.start()
		assert.NoError(t, err)
		time.Sleep(time.Second * 5)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := nodesService[1].host.discovery.start()
		assert.NoError(t, err)
		time.Sleep(time.Second * 5)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := nodesService[2].host.discovery.start()
		assert.NoError(t, err)
		time.Sleep(time.Second * 5)
	}()

	wg.Wait()

	// assert B and C can discover each other
	addrs := nodesService[1].host.p2pHost.Peerstore().Addrs(nodesService[2].host.id())
	fmt.Println("len(addrs): ", len(addrs))
	assert.NotEqual(t, 0, len(addrs))

	fmt.Println(nodesService[1].host.p2pHost.Network().Conns())
}
