// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_createRuntimeStorage(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	config.ChainSpec = NewTestGenesisRawFile(t, config)

	builder := nodeBuilder{}
	err := builder.initNode(config)
	require.NoError(t, err)

	stateSrvc, err := builder.createStateService(config)
	require.NoError(t, err)

	tests := []struct {
		name           string
		service        *state.Service
		expectedBaseDB *state.BaseState
		err            error
	}{
		{
			name:           "working example",
			service:        stateSrvc,
			expectedBaseDB: stateSrvc.Base,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builder.createRuntimeStorage(tt.service)
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.expectedBaseDB, got.BaseDB)
			assert.NotNil(t, got.LocalStorage)
			assert.NotNil(t, got.PersistentStorage)
		})
	}
}

func Test_createSystemService(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	config.ChainSpec = NewTestGenesisRawFile(t, config)

	builder := nodeBuilder{}
	err := builder.initNode(config)
	require.NoError(t, err)

	stateSrvc, err := builder.createStateService(config)
	require.NoError(t, err)

	type args struct {
		cfg     *types.SystemInfo
		service *state.Service
	}
	tests := []struct {
		name      string
		args      args
		expectNil bool
		err       error
	}{
		{
			name: "working_example",
			args: args{
				service: stateSrvc,
			},
			expectNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builder.createSystemService(tt.args.cfg, tt.args.service)
			assert.ErrorIs(t, err, tt.err)

			// TODO: change this check to assert.Equal after state.Service interface is implemented.
			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_newInMemoryDB(t *testing.T) {
	tests := []struct {
		name      string
		expectNil bool
		err       error
	}{
		{
			name:      "working example",
			expectNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newInMemoryDB()
			assert.ErrorIs(t, err, tt.err)

			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func newStateService(t *testing.T, ctrl *gomock.Controller) *state.Service {
	t.Helper()

	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	stateConfig := state.Config{
		Path:      t.TempDir(),
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}
	stateSrvc := state.NewService(stateConfig)
	stateSrvc.UseMemDB()
	genData, genTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(&genData, &genesisHeader, &genTrie)
	require.NoError(t, err)

	err = stateSrvc.SetupBase()
	require.NoError(t, err)

	genesisBABEConfig := &types.BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: []types.AuthorityRaw{},
		Randomness:         [32]byte{},
		SecondarySlots:     0,
	}
	epochState, err := state.NewEpochStateFromGenesis(stateSrvc.DB(), stateSrvc.Block, genesisBABEConfig)
	require.NoError(t, err)

	stateSrvc.Epoch = epochState

	var rtCfg wazero_runtime.Config

	rtCfg.Storage = rtstorage.NewTransactionalTrieState(&genTrie)

	rtCfg.CodeHash, err = stateSrvc.Storage.LoadCodeHash(nil)
	require.NoError(t, err)

	rtCfg.NodeStorage = runtime.NodeStorage{}

	rt, err := wazero_runtime.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	stateSrvc.Block.StoreRuntime(stateSrvc.Block.BestBlockHash(), rt)

	return stateSrvc
}
