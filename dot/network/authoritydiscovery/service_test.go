// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package authoritydiscovery

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestPeerID(t *testing.T) peer.ID {
	t.Helper()
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1) // Use -1 for random seed
	require.NoError(t, err)
	peerID, err := peer.IDFromPrivateKey(priv)
	require.NoError(t, err)
	return peerID
}

func createTestMultiaddr(t *testing.T, port int) ma.Multiaddr {
	t.Helper()
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	require.NoError(t, err)
	return addr
}

func TestService_GetAddressesByAuthorityID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan := make(chan ServiceToWorkerMsg, 10)
	service := NewService(msgChan)

	// Create test worker to handle messages
	worker := NewWorker(msgChan)
	go worker.Run(ctx)

	// Prepare test data
	authority := AuthorityID("test-authority-1")
	addr1 := createTestMultiaddr(t, 1234)
	addr2 := createTestMultiaddr(t, 5678)

	// Add test data to worker
	worker.AddAuthorityAddress(authority, addr1)
	worker.AddAuthorityAddress(authority, addr2)

	// Give worker time to process
	time.Sleep(20 * time.Millisecond) // Increased slightly for CI stability

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		addresses, err := service.GetAddressesByAuthorityID(ctx, authority)
		assert.NoError(t, err)
		assert.NotNil(t, addresses)
		assert.Len(t, addresses, 2)

		// Check if both addresses are in the slice
		found1, found2 := false, false
		for _, addr := range addresses {
			if addr.Equal(addr1) {
				found1 = true
			}
			if addr.Equal(addr2) {
				found2 = true
			}
		}
		assert.True(t, found1, "addr1 not found")
		assert.True(t, found2, "addr2 not found")
	})

	// Test non-existent authority
	t.Run("non-existent_authority", func(t *testing.T) {
		addresses, err := service.GetAddressesByAuthorityID(ctx, AuthorityID("unknown"))
		assert.NoError(t, err)
		assert.Nil(t, addresses)
	})

	// Test context cancellation
	t.Run("context_cancellation", func(t *testing.T) {
		cancelCtx, cancelFunc := context.WithCancel(ctx)
		cancelFunc() // Cancel immediately

		_, err := service.GetAddressesByAuthorityID(cancelCtx, authority)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	// Test closed service
	t.Run("closed_service", func(t *testing.T) {
		// Create a new service for this test
		closedServiceMsgChan := make(chan ServiceToWorkerMsg, 10)
		closedService := NewService(closedServiceMsgChan)
		closedService.Close()

		addresses, err := closedService.GetAddressesByAuthorityID(ctx, authority)
		assert.Error(t, err)
		assert.Equal(t, ErrServiceClosed, err)
		assert.Nil(t, addresses)
	})
}

func TestService_GetAuthorityIDsByPeerID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan := make(chan ServiceToWorkerMsg, 10)
	service := NewService(msgChan)

	// Create test worker to handle messages
	worker := NewWorker(msgChan)
	go worker.Run(ctx)

	// Prepare test data
	peerID := createTestPeerID(t)
	authority1 := AuthorityID("test-authority-1")
	authority2 := AuthorityID("test-authority-2")

	// Add test data to worker
	worker.AddPeerAuthority(peerID, authority1)
	worker.AddPeerAuthority(peerID, authority2)

	// Give worker time to process
	time.Sleep(20 * time.Millisecond) // Increased slightly for CI stability

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		authorities, err := service.GetAuthorityIDsByPeerID(ctx, peerID)
		assert.NoError(t, err)
		assert.NotNil(t, authorities)
		assert.Len(t, authorities, 2)

		// Check if both authorities are in the slice
		found1, found2 := false, false
		for _, auth := range authorities {
			if string(auth) == string(authority1) {
				found1 = true
			}
			if string(auth) == string(authority2) {
				found2 = true
			}
		}
		assert.True(t, found1, "authority1 not found")
		assert.True(t, found2, "authority2 not found")
	})

	// Test non-existent peer
	t.Run("non-existent_peer", func(t *testing.T) {
		unknownPeer := createTestPeerID(t)
		authorities, err := service.GetAuthorityIDsByPeerID(ctx, unknownPeer)
		assert.NoError(t, err)
		assert.Nil(t, authorities)
	})

	// Test context cancellation
	t.Run("context_cancellation", func(t *testing.T) {
		cancelCtx, cancelFunc := context.WithCancel(ctx)
		cancelFunc() // Cancel immediately

		_, err := service.GetAuthorityIDsByPeerID(cancelCtx, peerID)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	// Test closed service
	t.Run("closed_service", func(t *testing.T) {
		// Create a new service for this test
		closedServiceMsgChan := make(chan ServiceToWorkerMsg, 10)
		closedService := NewService(closedServiceMsgChan)
		closedService.Close()

		authorities, err := closedService.GetAuthorityIDsByPeerID(ctx, peerID)
		assert.Error(t, err)
		assert.Equal(t, ErrServiceClosed, err)
		assert.Nil(t, authorities)
	})
}

func TestService_Close(t *testing.T) {
	msgChan := make(chan ServiceToWorkerMsg, 10)
	service := NewService(msgChan)

	// First close should succeed
	service.Close()
	assert.True(t, service.closed, "service should be marked as closed")

	// Second close should be idempotent
	service.Close()
	assert.True(t, service.closed, "service should remain closed")
}

func TestService_ConcurrentAccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan := make(chan ServiceToWorkerMsg, 100) // Larger buffer for concurrent access
	service := NewService(msgChan)

	// Create test worker to handle messages
	worker := NewWorker(msgChan)
	go worker.Run(ctx)

	// Add test data
	authority := AuthorityID("test-authority")
	addr := createTestMultiaddr(t, 9999)
	worker.AddAuthorityAddress(authority, addr)

	peerID := createTestPeerID(t)
	worker.AddPeerAuthority(peerID, authority)

	// Give worker time to process
	time.Sleep(20 * time.Millisecond) // Increased slightly for CI stability

	// Test concurrent access to both methods
	var wg sync.WaitGroup
	numGoroutines := 20 // 10 for each method type

	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := service.GetAddressesByAuthorityID(ctx, authority)
			assert.NoError(t, err)
		}()

		go func() {
			defer wg.Done()
			_, err := service.GetAuthorityIDsByPeerID(ctx, peerID)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}

func TestService_TimeoutHandling(t *testing.T) {
	ctx := context.Background()
	msgChan := make(chan ServiceToWorkerMsg) // No buffer to force blocking
	service := NewService(msgChan)

	// Don't start a worker to simulate timeout scenario

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond) // Reduced timeout
	defer cancel()

	// This should timeout because no worker is processing messages
	_, err := service.GetAddressesByAuthorityID(timeoutCtx, AuthorityID("test"))
	assert.Error(t, err)
	assert.Contains(t,
		[]error{context.DeadlineExceeded, context.Canceled},
		err,
		"error should be context.DeadlineExceeded or context.Canceled",
	)
}

func TestWorker_MessageHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan := make(chan ServiceToWorkerMsg, 10)
	worker := NewWorker(msgChan)

	// Run worker in background
	go worker.Run(ctx)

	t.Run("GetAddressesByAuthorityID", func(t *testing.T) {
		authority := AuthorityID("test-authority")
		addr := createTestMultiaddr(t, 3333)
		worker.AddAuthorityAddress(authority, addr)
		time.Sleep(10 * time.Millisecond) // Give worker time to process add

		responseChan := make(chan *AddressesResponse, 1)
		msgChan <- GetAddressesByAuthorityID{
			AuthorityID: authority,
			Response:    responseChan,
		}

		select {
		case resp := <-responseChan:
			assert.NoError(t, resp.Err)
			assert.Len(t, resp.Addresses, 1)
			assert.True(t, resp.Addresses[0].Equal(addr))
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("GetAuthorityIDsByPeerID", func(t *testing.T) {
		peerID := createTestPeerID(t)
		authority := AuthorityID("test-authority")
		worker.AddPeerAuthority(peerID, authority)
		time.Sleep(10 * time.Millisecond) // Give worker time to process add

		responseChan := make(chan *AuthorityIDsResponse, 1)
		msgChan <- GetAuthorityIDsByPeerID{
			PeerID:   peerID,
			Response: responseChan,
		}

		select {
		case resp := <-responseChan:
			assert.NoError(t, resp.Err)
			assert.Len(t, resp.AuthorityIDs, 1)
			assert.Equal(t, string(authority), string(resp.AuthorityIDs[0]))
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for response")
		}
	})
}

func TestWorker_ContextCancellation(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	msgChan := make(chan ServiceToWorkerMsg, 1) // Small buffer
	worker := NewWorker(msgChan)

	workerDone := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(workerDone)
	}()

	cancelFunc()

	// Give worker time to exit
	select {
	case <-workerDone:
		// Worker exited cleanly
	case <-time.After(200 * time.Millisecond): // Increased timeout
		t.Fatal("worker did not stop after context cancellation")
	}

	// Try to send a message - worker should have stopped
	responseChan := make(chan *AddressesResponse, 1)
	msg := GetAddressesByAuthorityID{
		AuthorityID: AuthorityID("test"),
		Response:    responseChan,
	}

	// This send might block if the channel was already full and worker stopped,
	// or succeed if worker processed an item before stopping.
	// The key is no response should come.
	select {
	case msgChan <- msg:
		// Message sent (worker might have processed one last item or channel had space)
		// but no response expected as worker should be stopping/stopped.
		select {
		case <-responseChan:
			t.Fatal("unexpected response from stopped worker")
		case <-time.After(100 * time.Millisecond):
			// Expected: no response
		}
	case <-time.After(100 * time.Millisecond):
		// Expected: send might block or fail if channel is unbuffered/full and worker stopped.
		// This is acceptable.
	}
}

func TestWorker_UniqueValues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan := make(chan ServiceToWorkerMsg, 10)
	worker := NewWorker(msgChan)

	// Run worker in background
	go worker.Run(ctx)

	// Test unique addresses for authority
	t.Run("unique_addresses", func(t *testing.T) {
		authority := AuthorityID("test-authority")
		addr := createTestMultiaddr(t, 4444)

		// Add the same address multiple times
		worker.AddAuthorityAddress(authority, addr)
		worker.AddAuthorityAddress(authority, addr)
		worker.AddAuthorityAddress(authority, addr)
		time.Sleep(10 * time.Millisecond) // Give worker time to process adds

		responseChan := make(chan *AddressesResponse, 1)
		msgChan <- GetAddressesByAuthorityID{
			AuthorityID: authority,
			Response:    responseChan,
		}

		select {
		case resp := <-responseChan:
			assert.NoError(t, resp.Err)
			assert.Len(t, resp.Addresses, 1, "duplicate addresses should not be stored")
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for response")
		}
	})

	// Test unique authorities for peer
	t.Run("unique_authorities", func(t *testing.T) {
		peerID := createTestPeerID(t)
		authority := AuthorityID("test-authority")

		// Add the same authority multiple times
		worker.AddPeerAuthority(peerID, authority)
		worker.AddPeerAuthority(peerID, authority)
		worker.AddPeerAuthority(peerID, authority)
		time.Sleep(10 * time.Millisecond) // Give worker time to process adds

		responseChan := make(chan *AuthorityIDsResponse, 1)
		msgChan <- GetAuthorityIDsByPeerID{
			PeerID:   peerID,
			Response: responseChan,
		}

		select {
		case resp := <-responseChan:
			assert.NoError(t, resp.Err)
			assert.Len(t, resp.AuthorityIDs, 1, "duplicate authorities should not be stored")
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for response")
		}
	})
}

func TestIntegration_ServiceAndWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan := make(chan ServiceToWorkerMsg, 10)
	service := NewService(msgChan)
	worker := NewWorker(msgChan)

	go worker.Run(ctx)

	// Set up test data
	authority1 := AuthorityID("authority-1")
	authority2 := AuthorityID("authority-2")
	addr1 := createTestMultiaddr(t, 1111)
	addr2 := createTestMultiaddr(t, 2222)
	peerID1 := createTestPeerID(t)
	peerID2 := createTestPeerID(t)

	// Add relationships
	worker.AddAuthorityAddress(authority1, addr1)
	worker.AddAuthorityAddress(authority1, addr2)
	worker.AddAuthorityAddress(authority2, addr2)

	worker.AddPeerAuthority(peerID1, authority1)
	worker.AddPeerAuthority(peerID2, authority1)
	worker.AddPeerAuthority(peerID2, authority2)

	time.Sleep(20 * time.Millisecond) // Increased slightly for CI stability

	// Test complex queries
	t.Run("multiple_addresses_for_authority", func(t *testing.T) {
		addresses, err := service.GetAddressesByAuthorityID(ctx, authority1)
		assert.NoError(t, err)
		assert.Len(t, addresses, 2)
	})

	t.Run("multiple_authorities_for_peer", func(t *testing.T) {
		authorities, err := service.GetAuthorityIDsByPeerID(ctx, peerID2)
		assert.NoError(t, err)
		assert.Len(t, authorities, 2)
	})

	t.Run("single_authority_for_peer", func(t *testing.T) {
		authorities, err := service.GetAuthorityIDsByPeerID(ctx, peerID1)
		assert.NoError(t, err)
		assert.Len(t, authorities, 1)
		assert.Equal(t, string(authority1), string(authorities[0]))
	})
}
