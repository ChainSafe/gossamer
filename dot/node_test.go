// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/database/badger"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitNode(t *testing.T) {
	cfg := NewWestendDevConfig(t)
	cfg.Init.Genesis = NewTestGenesisRawFile(t, cfg)
	tests := []struct {
		name   string
		config *Config
		err    error
	}{
		{
			name:   "test config",
			config: cfg,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitNode(tt.config)
			assert.ErrorIs(t, err, tt.err)
			// confirm InitNode has created database dir
			registry := filepath.Join(tt.config.Global.BasePath, utils.DefaultDatabaseDir, "KEYREGISTRY")
			_, err = os.Stat(registry)
			require.NoError(t, err)
		})
	}
}

func TestLoadGlobalNodeName(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	var settings badger.Settings
	settings.WithPath(filepath.Join(basePath, "db"))
	db, err := badger.New(settings)
	require.NoError(t, err)

	basestate := state.NewBaseState(db)
	basestate.Set(common.NodeNameKey, []byte(`nodeName`))

	err = db.Close()
	require.NoError(t, err)

	tests := []struct {
		name         string
		basepath     string
		wantNodename string
		err          error
	}{
		{
			name:         "working example",
			basepath:     basePath,
			wantNodename: "nodeName",
		},
		{
			name:     "wrong basepath test",
			basepath: t.TempDir(),
			err:      errors.New("key not found: 0x6e6f64655f6e616d65"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotNodename, err := LoadGlobalNodeName(tt.basepath)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantNodename, gotNodename)
		})
	}
}

func setConfigTestDefaults(t *testing.T, cfg *network.Config) {
	t.Helper()
	ctrl := gomock.NewController(t)

	if cfg == nil {
		cfg = &network.Config{
			BasePath:     t.TempDir(),
			NoBootstrap:  true,
			NoMDNS:       true,
			LogLvl:       log.Warn,
			SlotDuration: time.Second,
		}
	}
	if cfg.BlockState == nil {
		blockstate := NewMockBlockState(ctrl)

		cfg.BlockState = blockstate
	}

	cfg.SlotDuration = time.Second

	if cfg.Telemetry == nil {
		telemetryMock := NewMockTelemetry(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()
		cfg.Telemetry = telemetryMock
	}
}

func TestNodeInitialized(t *testing.T) {
	cfg := NewWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	nodeInstance := nodeBuilder{}
	err := nodeInstance.initNode(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		basepath string
		want     bool
	}{
		{
			name:     "blank base path",
			basepath: "",
			want:     false,
		},
		{
			name:     "working example",
			basepath: cfg.Global.BasePath,
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNodeInitialised(tt.basepath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func initKeystore(t *testing.T, cfg *Config) (
	globalKeyStore *keystore.GlobalKeystore, err error) {
	ks := keystore.NewGlobalKeystore()

	sr25519KeyRing, err := keystore.NewSr25519Keyring()
	if err != nil {
		return nil, fmt.Errorf("creating sr25519 keyring: %w", err)
	}

	// load built-in test keys if specified by `cfg.Account.Key`
	err = keystore.LoadKeystore(cfg.Account.Key, ks.Acco, sr25519KeyRing)
	require.NoError(t, err)

	err = keystore.LoadKeystore(cfg.Account.Key, ks.Babe, sr25519KeyRing)
	require.NoError(t, err)

	ed25519KeyRing, err := keystore.NewEd25519Keyring()
	if err != nil {
		return nil, fmt.Errorf("creating ed25519 keyring: %w", err)
	}

	err = keystore.LoadKeystore(cfg.Account.Key, ks.Gran, ed25519KeyRing)
	require.NoError(t, err)

	// if authority node, should have at least 1 key in keystore
	if cfg.Core.Roles == common.AuthorityRole && (ks.Babe.Size() == 0 || ks.Gran.Size() == 0) {
		return nil, ErrNoKeysProvided
	}

	return ks, nil
}

func TestNode_StartStop(t *testing.T) {
	serviceRegistryLogger := logger.New(log.AddContext("pkg", "services"))
	type fields struct {
		Name          string
		Services      *services.ServiceRegistry
		started       chan struct{}
		metricsServer *metrics.Server
	}
	tests := []struct {
		name   string
		fields fields
		err    error
	}{
		{
			name: "base_case",
			fields: fields{
				Name:     "Node",
				Services: services.NewServiceRegistry(serviceRegistryLogger),
				started:  make(chan struct{}),
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				Name:            tt.fields.Name,
				ServiceRegistry: tt.fields.Services,
				started:         tt.fields.started,
				metricsServer:   tt.fields.metricsServer,
			}
			go func() {
				<-n.started
				n.Stop()
			}()

			err := n.Start()
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
