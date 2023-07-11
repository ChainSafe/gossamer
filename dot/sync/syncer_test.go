// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfgBuilder func(ctrl *gomock.Controller) *Config
		want       *Service
		err        error
	}{
		{
			name: "working_example",
			cfgBuilder: func(ctrl *gomock.Controller) *Config {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetFinalisedNotifierChannel().
					Return(make(chan *types.FinalisationInfo))
				return &Config{
					BlockState: blockState,
				}
			},
			want: &Service{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			config := tt.cfgBuilder(ctrl)

			got, err := NewService(config)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestService_HandleBlockAnnounce(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")
	const somePeer = peer.ID("abc")

	block1AnnounceHeader := types.NewHeader(common.Hash{}, common.Hash{},
		common.Hash{}, 1, scale.VaryingDataTypeSlice{})
	block2AnnounceHeader := types.NewHeader(common.Hash{}, common.Hash{},
		common.Hash{}, 2, scale.VaryingDataTypeSlice{})

	testCases := map[string]struct {
		serviceBuilder      func(ctrl *gomock.Controller) *Service
		peerID              peer.ID
		blockAnnounceHeader *types.Header
		errWrapped          error
		errMessage          string
	}{
		"best_block_header_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().Return(nil, errTest)
				return &Service{
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "best block header: test error",
		},
		"number_smaller_than_best_block_number_get_hash_by_number_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{}, errTest)

				return &Service{
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "get block hash by number: test error",
		},
		"number_smaller_than_best_block_number_and_same_hash": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).Return(block1AnnounceHeader.Hash(), nil)
				return &Service{
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
		},
		"number_smaller_than_best_block_number_get_highest_finalised_header_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{2}, nil)
				blockState.EXPECT().GetHighestFinalisedHeader().Return(nil, errTest)
				return &Service{
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "get highest finalised header: test error",
		},
		"number_smaller_than_best_block_announced_number_equaks_finalised_number": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)

				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).
					Return(common.Hash{2}, nil) // other hash than someHash
				finalisedBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				network := NewMockNetwork(ctrl)
				network.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadBlockAnnouncementValue,
					Reason: peerset.BadBlockAnnouncementReason,
				}, somePeer)
				return &Service{
					blockState: blockState,
					network:    network,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errPeerOnInvalidFork,
			errMessage:          "peer is on an invalid fork: for peer ZiCa and block number 1",
		},
		"number_smaller_than_best_block_number_and_finalised_number_bigger_than_number": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).
					Return(common.Hash{2}, nil) // other hash than someHash
				finalisedBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				network := NewMockNetwork(ctrl)
				network.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadBlockAnnouncementValue,
					Reason: peerset.BadBlockAnnouncementReason,
				}, somePeer)
				return &Service{
					blockState: blockState,
					network:    network,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errPeerOnInvalidFork,
			errMessage:          "peer is on an invalid fork: for peer ZiCa and block number 1",
		},
		"number_smaller_than_best_block_number_and_" +
			"finalised_number_smaller_than_number_and_" +
			"has_header_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 3}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(2)).
					Return(common.Hash{5, 1, 2}, nil) // other hash than block2AnnounceHeader hash
				finalisedBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				blockState.EXPECT().HasHeader(block2AnnounceHeader.Hash()).Return(false, errTest)
				return &Service{
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "while checking if header exists: test error",
		},
		"number_smaller_than_best_block_number_and_" +
			"finalised_number_smaller_than_number_and_" +
			"has_the_hash": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 3}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(2)).
					Return(common.Hash{2}, nil) // other hash than someHash
				finalisedBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				blockState.EXPECT().HasHeader(block2AnnounceHeader.Hash()).Return(true, nil)
				return &Service{
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
		},
		"number_bigger_than_best_block_number_added_in_disjoint_set_with_success": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {

				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				chainSyncMock := NewMockChainSync(ctrl)

				expectedAnnouncedBlock := announcedBlock{
					who:    somePeer,
					header: block2AnnounceHeader,
				}

				chainSyncMock.EXPECT().onBlockAnnounce(expectedAnnouncedBlock).Return(nil)

				return &Service{
					blockState: blockState,
					chainSync:  chainSyncMock,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
		},
	}

	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			service := tt.serviceBuilder(ctrl)

			blockAnnounceMessage := &network.BlockAnnounceMessage{
				ParentHash:     tt.blockAnnounceHeader.ParentHash,
				Number:         tt.blockAnnounceHeader.Number,
				StateRoot:      tt.blockAnnounceHeader.StateRoot,
				ExtrinsicsRoot: tt.blockAnnounceHeader.ExtrinsicsRoot,
				Digest:         tt.blockAnnounceHeader.Digest,
				BestBlock:      true,
			}
			err := service.HandleBlockAnnounce(tt.peerID, blockAnnounceMessage)
			assert.ErrorIs(t, err, tt.errWrapped)
			if tt.errWrapped != nil {
				assert.EqualError(t, err, tt.errMessage)
			}
		})
	}
}

func Test_Service_HandleBlockAnnounceHandshake(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	chainSync := NewMockChainSync(ctrl)
	chainSync.EXPECT().setPeerHead(peer.ID("peer"), common.Hash{1}, uint(2))

	service := Service{
		chainSync: chainSync,
	}

	message := &network.BlockAnnounceHandshake{
		BestBlockHash:   common.Hash{1},
		BestBlockNumber: 2,
	}

	err := service.HandleBlockAnnounceHandshake(peer.ID("peer"), message)
	require.Nil(t, err)
}

func TestService_IsSynced(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		serviceBuilder func(ctrl *gomock.Controller) Service
		synced         bool
	}{
		"tip": {
			serviceBuilder: func(ctrl *gomock.Controller) Service {
				chainSync := NewMockChainSync(ctrl)
				chainSync.EXPECT().getSyncMode().Return(tip)
				return Service{
					chainSync: chainSync,
				}
			},
			synced: true,
		},
		"not_tip": {
			serviceBuilder: func(ctrl *gomock.Controller) Service {
				chainSync := NewMockChainSync(ctrl)
				chainSync.EXPECT().getSyncMode().Return(bootstrap)
				return Service{
					chainSync: chainSync,
				}
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			service := testCase.serviceBuilder(ctrl)

			synced := service.IsSynced()

			assert.Equal(t, testCase.synced, synced)
		})
	}
}

func TestService_Start(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	var allCalled sync.WaitGroup

	chainSync := NewMockChainSync(ctrl)
	allCalled.Add(1)
	chainSync.EXPECT().start().DoAndReturn(func() {
		allCalled.Done()
	})

	service := Service{
		chainSync: chainSync,
	}

	err := service.Start()
	allCalled.Wait()
	assert.NoError(t, err)
}

func TestService_Stop(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	chainSync := NewMockChainSync(ctrl)
	chainSync.EXPECT().stop()
	service := &Service{
		chainSync: chainSync,
	}

	err := service.Stop()
	assert.NoError(t, err)
}

func Test_reverseBlockData(t *testing.T) {
	t.Parallel()

	type args struct {
		data []*types.BlockData
	}
	tests := []struct {
		name     string
		args     args
		expected args
	}{
		{
			name: "working_example",
			args: args{data: []*types.BlockData{
				{
					Hash: common.MustHexToHash("0x01"),
				},
				{
					Hash: common.MustHexToHash("0x02"),
				}}},
			expected: args{data: []*types.BlockData{{
				Hash: common.MustHexToHash("0x02"),
			}, {
				Hash: common.MustHexToHash("0x01"),
			}},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reverseBlockData(tt.args.data)
			assert.Equal(t, tt.expected.data, tt.args.data)
		})
	}
}

func TestService_HighestBlock(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	chainSync := NewMockChainSync(ctrl)
	chainSync.EXPECT().getHighestBlock().Return(uint(2), nil)

	service := &Service{
		chainSync: chainSync,
	}
	highestBlock := service.HighestBlock()
	const expected = uint(2)
	assert.Equal(t, expected, highestBlock)
}
