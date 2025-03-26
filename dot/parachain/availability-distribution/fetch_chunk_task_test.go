package availabilitydistribution

import (
	"errors"
	"sync"
	"testing"
	"time"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

type responder func(t *testing.T, request *networkbridgemessages.OutgoingRequest)

func TestFetchChunkTask(t *testing.T) {
	var (
		task       *fetchChunkTask
		overseerCh chan any

		leaf       = common.MustHexToHash("0x42")
		chunkIndex = uint32(23)
		ourIndex   = parachaintypes.ValidatorIndex(4)

		core = &parachaintypes.OccupiedCore{
			CandidateHash: common.MustHexToHash("0x1337"),
			CandidateDescriptor: parachaintypes.CandidateDescriptor{
				ErasureRoot: common.MustHexToHash("0xABCD"),
			},
			GroupResponsible: parachaintypes.GroupIndex(0),
		}
		expectedCandidateHash = parachaintypes.CandidateHash{Value: core.CandidateHash}

		validator1 = parachaintypes.AuthorityDiscoveryID{1}
		validator2 = parachaintypes.AuthorityDiscoveryID{2}
		validator3 = parachaintypes.AuthorityDiscoveryID{3}

		sessionInfo = &SessionInfo{
			SessionIndex: parachaintypes.SessionIndex(1),
			ValidatorGroups: [][]parachaintypes.AuthorityDiscoveryID{
				{validator1, validator2, validator3},
			},
			OurIndex: ourIndex,
		}

		group = sessionInfo.ValidatorGroups[core.GroupResponsible]
	)

	setup := func(t *testing.T, onTermination taskTerminationHandler) {
		overseerCh = make(chan any)

		task = newFetchChunkTask(
			leaf,
			chunkIndex,
			sessionInfo,
			core,
			overseerCh,
			onTermination,
		)
		require.NotNil(t, task)
	}

	respondWith := func(
		t *testing.T,
		msg any,
		expectedRecipient parachaintypes.RecipientID,
		responder responder,
	) {
		sendRequests, ok := msg.(networkbridgemessages.SendRequests)
		require.True(t, ok)
		require.Equal(t, 1, len(sendRequests.Requests))

		request := sendRequests.Requests[0]
		require.Equal(t, expectedRecipient, request.Recipient)

		cfRequest, ok := request.Payload.(*networkbridgemessages.ChunkFetchingRequest)
		require.True(t, ok)
		require.Equal(t, expectedCandidateHash, cfRequest.CandidateHash)
		require.Equal(t, ourIndex, cfRequest.Index)

		responder(t, request)
	}

	t.Run("retries_requests", func(t *testing.T) {
		setup(t, func(
			candidateHash parachaintypes.CandidateHash,
			reason taskTerminationReason,
		) {
			require.Equal(t, expectedCandidateHash, candidateHash)
			success, ok := reason.(taskSucceeded)
			require.True(t, ok)
			require.Equal(t, success.badValidators, []parachaintypes.AuthorityDiscoveryID{validator3, validator2})
		})

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			task.run()
			wg.Done()
		}()

		// validator3 is tried first, does not deliver
		select {
		case msg := <-overseerCh:
			respondWith(t, msg, validator3, func(t *testing.T, request *networkbridgemessages.OutgoingRequest) {
				request.Result <- networkbridgemessages.ReqRespResult{
					Response: nil,
					Error:    errors.New("network failure"),
				}
			})
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout")
		}

		// validator2 is tried second, does not deliver
		select {
		case msg := <-overseerCh:
			response := networkbridgemessages.ChunkFetchingResponse{}
			require.NoError(t, response.SetValue(networkbridgemessages.NoSuchChunk{}))

			respondWith(t, msg, validator2, func(t *testing.T, request *networkbridgemessages.OutgoingRequest) {
				request.Result <- networkbridgemessages.ReqRespResult{
					Response: &response,
					Error:    nil,
				}
			})
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout")
		}

		// validator1 is tried last, delivers
		select {
		case msg := <-overseerCh:
			response := networkbridgemessages.ChunkFetchingResponse{}
			require.NoError(t, response.SetValue(networkbridgemessages.ChunkResponse{
				Chunk: []byte{0x01, 0x02},
				Index: chunkIndex,
				Proof: [][]byte{{0x03, 0x04}}, // FIXME Use real values once chunk validation is implemented.
			}))

			respondWith(t, msg, validator1, func(t *testing.T, request *networkbridgemessages.OutgoingRequest) {
				request.Result <- networkbridgemessages.ReqRespResult{
					Response: &response,
					Error:    nil,
				}
			})
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout")
		}

		select {
		case msg := <-overseerCh:
			storeChunkMsg, ok := msg.(availabilitystore.StoreChunk)
			require.True(t, ok)
			require.Equal(t, expectedCandidateHash, storeChunkMsg.CandidateHash)
			require.Equal(t, []byte{0x01, 0x02}, storeChunkMsg.Chunk.Chunk)
			require.Equal(t, chunkIndex, storeChunkMsg.Chunk.Index)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout")
		}

		wg.Wait()
	})

	t.Run("group_exhausted", func(t *testing.T) {
		setup(t, func(
			candidateHash parachaintypes.CandidateHash,
			reason taskTerminationReason,
		) {
			require.Equal(t, expectedCandidateHash, candidateHash)
			require.Equal(t, taskFailed{}, reason)
		})

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			task.run()
			wg.Done()
		}()

		// each validator responds with NoSuchChunk
		for i := len(group) - 1; i >= 0; i-- {
			validator := group[i]
			select {
			case msg := <-overseerCh:
				response := networkbridgemessages.ChunkFetchingResponse{}
				require.NoError(t, response.SetValue(networkbridgemessages.NoSuchChunk{}))

				respondWith(t, msg, validator, func(t *testing.T, request *networkbridgemessages.OutgoingRequest) {
					request.Result <- networkbridgemessages.ReqRespResult{
						Response: &response,
						Error:    nil,
					}
				})
			case <-time.After(100 * time.Millisecond):
				t.Fatal("timeout")
			}
		}

		wg.Wait()
	})

	t.Run("task_cancellation", func(t *testing.T) {
		setup(t, func(
			candidateHash parachaintypes.CandidateHash,
			reason taskTerminationReason,
		) {
			require.Equal(t, expectedCandidateHash, candidateHash)
			require.Equal(t, taskCancelled{}, reason)
		})

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			task.run()
			wg.Done()
		}()

		// Make the task cancel itself by removing the only leaf it is relevant for.
		task.removeLeaves([]common.Hash{leaf})

		select {
		case msg := <-overseerCh:
			response := networkbridgemessages.ChunkFetchingResponse{}
			require.NoError(t, response.SetValue(networkbridgemessages.NoSuchChunk{}))

			respondWith(t, msg, validator3, func(t *testing.T, request *networkbridgemessages.OutgoingRequest) {
				// Simulate network latency and, more pragmatically,
				// hopefully avoid flaky tests caused by race conditions.
				time.Sleep(10 * time.Millisecond)
				require.True(t, request.IsCancelled())
				close(request.Result)
			})
		case <-time.After(100 * time.Millisecond):
			// Not a test failure in this case.
			// The task might have been cancelled before any requests were sent.
			// The main assertion is in the termination handler.
		}

		wg.Wait()
	})

	t.Run("task_cancellation_after_request_sent", func(t *testing.T) {
		setup(t, func(
			candidateHash parachaintypes.CandidateHash,
			reason taskTerminationReason,
		) {
			require.Equal(t, expectedCandidateHash, candidateHash)
			require.Equal(t, taskCancelled{}, reason)
		})

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			task.run()
			wg.Done()
		}()

		select {
		case msg := <-overseerCh:
			response := networkbridgemessages.ChunkFetchingResponse{}
			require.NoError(t, response.SetValue(networkbridgemessages.NoSuchChunk{}))

			respondWith(t, msg, validator3, func(t *testing.T, request *networkbridgemessages.OutgoingRequest) {
				// Make the task cancel itself *after* it has sent a network request.
				// It should terminate without waiting for a response.
				task.removeLeaves([]common.Hash{leaf})
			})
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout")
		}

		wg.Wait()
	})
}
