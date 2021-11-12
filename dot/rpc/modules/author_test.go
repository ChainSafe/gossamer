// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuthorModule_HasSessionKey_WhenScaleDataEmptyOrNil(t *testing.T) {
	keys := "0x00"
	runtimeInstance := wasmer.NewTestInstance(t, runtime.NODE_RUNTIME)

	coremockapi := new(apimocks.CoreAPI)

	decodeSessionKeysMock := coremockapi.On("DecodeSessionKeys", mock.AnythingOfType("[]uint8"))
	decodeSessionKeysMock.Run(func(args mock.Arguments) {
		b := args.Get(0).([]byte)
		dec, err := runtimeInstance.DecodeSessionKeys(b)
		decodeSessionKeysMock.ReturnArguments = []interface{}{dec, err}
	})

	module := &AuthorModule{
		coreAPI: coremockapi,
		logger:  log.New(log.SetWriter(io.Discard)),
	}

	req := &HasSessionKeyRequest{
		PublicKeys: keys,
	}

	var res HasSessionKeyResponse
	err := module.HasSessionKeys(nil, req, &res)
	require.NoError(t, err)
	require.False(t, bool(res))

	coremockapi.AssertCalled(t, "DecodeSessionKeys", mock.AnythingOfType("[]uint8"))
}

func TestAuthorModule_HasSessionKey_WhenRuntimeFails(t *testing.T) {
	coremockapi := new(apimocks.CoreAPI)
	coremockapi.On("DecodeSessionKeys", mock.AnythingOfType("[]uint8")).Return(nil, errors.New("problems with runtime"))

	module := &AuthorModule{
		coreAPI: coremockapi,
		logger:  log.New(log.SetWriter(io.Discard)),
	}

	req := &HasSessionKeyRequest{
		PublicKeys: "0x00",
	}

	var res HasSessionKeyResponse
	err := module.HasSessionKeys(nil, req, &res)
	require.Error(t, err, "problems with runtime")
	require.False(t, bool(res))
}

func TestAuthorModule_HasSessionKey_WhenThereIsNoKeys(t *testing.T) {
	keys := "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d34309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026"
	runtimeInstance := wasmer.NewTestInstance(t, runtime.NODE_RUNTIME)

	coremockapi := new(apimocks.CoreAPI)
	coremockapi.On("HasKey", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(false, nil)

	decodeSessionKeysMock := coremockapi.On("DecodeSessionKeys", mock.AnythingOfType("[]uint8"))
	decodeSessionKeysMock.Run(func(args mock.Arguments) {
		b := args.Get(0).([]byte)
		dec, err := runtimeInstance.DecodeSessionKeys(b)
		decodeSessionKeysMock.ReturnArguments = []interface{}{dec, err}
	})

	module := &AuthorModule{
		coreAPI: coremockapi,
		logger:  log.New(log.SetWriter(io.Discard)),
	}

	req := &HasSessionKeyRequest{
		PublicKeys: keys,
	}

	var res HasSessionKeyResponse
	err := module.HasSessionKeys(nil, req, &res)
	require.NoError(t, err)
	require.False(t, bool(res))

	coremockapi.AssertCalled(t, "DecodeSessionKeys", mock.AnythingOfType("[]uint8"))
	coremockapi.AssertCalled(t, "HasKey", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
	coremockapi.AssertNumberOfCalls(t, "HasKey", 1)
}

func TestAuthorModule_HasSessionKey(t *testing.T) {
	globalStore := keystore.NewGlobalKeystore()

	coremockapi := new(apimocks.CoreAPI)
	mockInsertKey := coremockapi.On("InsertKey", mock.AnythingOfType("*sr25519.Keypair"), mock.AnythingOfType("string")).Return(nil)
	mockInsertKey.Run(func(args mock.Arguments) {
		kp := args.Get(0).(*sr25519.Keypair)
		globalStore.Acco.Insert(kp)
	})

	mockHasKey := coremockapi.On("HasKey", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
	mockHasKey.Run(func(args mock.Arguments) {
		pubKeyHex := args.Get(0).(string)
		keyType := args.Get(1).(string)

		ok, err := keystore.HasKey(pubKeyHex, keyType, globalStore.Acco)
		mockHasKey.ReturnArguments = []interface{}{ok, err}
	})

	keys := "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d34309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026"
	runtimeInstance := wasmer.NewTestInstance(t, runtime.NODE_RUNTIME)

	decodeSessionKeysMock := coremockapi.On("DecodeSessionKeys", mock.AnythingOfType("[]uint8"))
	decodeSessionKeysMock.Run(func(args mock.Arguments) {
		b := args.Get(0).([]byte)
		dec, err := runtimeInstance.DecodeSessionKeys(b)
		decodeSessionKeysMock.ReturnArguments = []interface{}{dec, err}
	})

	module := &AuthorModule{
		coreAPI: coremockapi,
		logger:  log.New(log.SetWriter(io.Discard)),
	}

	req := &HasSessionKeyRequest{
		PublicKeys: keys,
	}

	err := module.InsertKey(nil, &KeyInsertRequest{
		Type:      "babe",
		Seed:      "0xfec0f475b818470af5caf1f3c1b1558729961161946d581d2755f9fb566534f8",
		PublicKey: "0x34309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026",
	}, nil)
	coremockapi.AssertCalled(t, "InsertKey", mock.AnythingOfType("*sr25519.Keypair"), mock.AnythingOfType("string"))
	require.NoError(t, err)
	require.Equal(t, 1, globalStore.Acco.Size())

	err = module.InsertKey(nil, &KeyInsertRequest{
		Type:      "babe",
		Seed:      "0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a",
		PublicKey: "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, 2, globalStore.Acco.Size())

	var res HasSessionKeyResponse
	err = module.HasSessionKeys(nil, req, &res)
	require.NoError(t, err)
	require.True(t, bool(res))
}

func TestAuthorModule_SubmitExtrinsic(t *testing.T) {
	errMockCoreAPI := &apimocks.CoreAPI{}
	errMockCoreAPI.On("HandleSubmittedExtrinsic", mock.AnythingOfType("types.Extrinsic")).Return(fmt.Errorf("some error"))

	mockCoreAPI := &apimocks.CoreAPI{}
	mockCoreAPI.On("HandleSubmittedExtrinsic", mock.AnythingOfType("types.Extrinsic")).Return(nil)

	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	var testExt = common.MustHexToBytes("0x410284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01f8efbe48487e57a22abf7e3acd491b7f3528a33a111b1298601554863d27eb129eaa4e718e1365414ff3d028b62bebc651194c6b5001e5c2839b982757e08a8c0000000600ff8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a480b00c465f14670")

	// invalid transaction (above tx, with last byte changed)
	//nolint
	var testInvalidExt = []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 143}

	type fields struct {
		logger     log.LeveledLogger
		coreAPI    CoreAPI
		txStateAPI TransactionStateAPI
	}
	type args struct {
		r   *http.Request
		req *Extrinsic
		res *ExtrinsicHashResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantRes ExtrinsicHashResponse
	}{
		{
			name: "HexToBytes error",
			args: args{
				req: &Extrinsic{fmt.Sprintf("%x", "1")},
				res: new(ExtrinsicHashResponse),
			},
			wantErr: true,
			wantRes: ExtrinsicHashResponse(""),
		},
		{
			name: "HandleSubmittedExtrinsic error",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: errMockCoreAPI,
			},
			args: args{
				req: &Extrinsic{fmt.Sprintf("0x%x", testInvalidExt)},
				res: new(ExtrinsicHashResponse),
			},
			wantErr: true,
			wantRes: ExtrinsicHashResponse(types.Extrinsic(testInvalidExt).Hash().String()),
		},
		{
			name: "happy path",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: mockCoreAPI,
			},
			args: args{
				req: &Extrinsic{fmt.Sprintf("0x%x", testExt)},
				res: new(ExtrinsicHashResponse),
			},
			wantRes: ExtrinsicHashResponse(types.Extrinsic(testExt).Hash().String()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := &AuthorModule{
				logger:     tt.fields.logger,
				coreAPI:    tt.fields.coreAPI,
				txStateAPI: tt.fields.txStateAPI,
			}
			if err := am.SubmitExtrinsic(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("AuthorModule.SubmitExtrinsic() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.wantRes, *tt.args.res); diff != "" {
				t.Errorf("unexpected response: %s", diff)
			}
		})
	}
}

func TestAuthorModule_PendingExtrinsics(t *testing.T) {
	emptyMockTransactionStateAPI := &apimocks.TransactionStateAPI{}
	emptyMockTransactionStateAPI.On("Pending").Return([]*transaction.ValidTransaction{})

	mockTransactionStateAPI := &apimocks.TransactionStateAPI{}
	mockTransactionStateAPI.On("Pending").Return([]*transaction.ValidTransaction{
		{
			Extrinsic: types.NewExtrinsic([]byte("someExtrinsic")),
		},
		{
			Extrinsic: types.NewExtrinsic([]byte("someExtrinsic1")),
		},
	})

	type fields struct {
		logger     log.LeveledLogger
		coreAPI    CoreAPI
		txStateAPI TransactionStateAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res *PendingExtrinsicsResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantRes PendingExtrinsicsResponse
	}{
		{
			name: "no pending",
			fields: fields{
				logger:     log.New(log.SetWriter(io.Discard)),
				txStateAPI: emptyMockTransactionStateAPI,
			},
			args: args{
				res: new(PendingExtrinsicsResponse),
			},
			wantRes: PendingExtrinsicsResponse{},
		},
		{
			name: "two pending",
			fields: fields{
				logger:     log.New(log.SetWriter(io.Discard)),
				txStateAPI: mockTransactionStateAPI,
			},
			args: args{
				res: new(PendingExtrinsicsResponse),
			},
			wantRes: PendingExtrinsicsResponse{
				common.BytesToHex([]byte("someExtrinsic")),
				common.BytesToHex([]byte("someExtrinsic1")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &AuthorModule{
				logger:     tt.fields.logger,
				coreAPI:    tt.fields.coreAPI,
				txStateAPI: tt.fields.txStateAPI,
			}
			if err := cm.PendingExtrinsics(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("AuthorModule.PendingExtrinsics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.wantRes, *tt.args.res); diff != "" {
				t.Errorf("unexpected response: %s", diff)
			}
		})
	}
}

func TestAuthorModule_InsertKey(t *testing.T) {
	mockCoreAPI := &apimocks.CoreAPI{}
	mockCoreAPI.On("InsertKey", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	type fields struct {
		logger     log.LeveledLogger
		coreAPI    CoreAPI
		txStateAPI TransactionStateAPI
	}
	type args struct {
		r   *http.Request
		req *KeyInsertRequest
		res *KeyInsertResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: mockCoreAPI,
			},
			args: args{
				req: &KeyInsertRequest{
					"babe",
					"0x6246ddf254e0b4b4e7dffefc8adf69d212b98ac2b579c362b473fec8c40b4c0a",
					"0xdad5131003242c37c227f744f82118dd59a24b949ae264a93d949100738c196c",
				},
			},
		},
		{
			name: "happy path, gran keytype",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: mockCoreAPI,
			},
			args: args{
				req: &KeyInsertRequest{
					"gran",
					"0xb48004c6e1625282313b07d1c9950935e86894a2e4f21fb1ffee9854d180c781",
					"0xa7d6507d59f8871b8f1a0f2c32e219adfacff4c9fcb05b0b2d8ebd6a65c88ee6",
				},
			},
		},
		{
			name: "invalid key",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: mockCoreAPI,
			},
			args: args{
				req: &KeyInsertRequest{"babe",
					"0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309",
					"0x0000000000000000000000000000000000000000000000000000000000000000",
				},
			},
			wantErr: true,
		},
		{
			name: "unknown key",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: mockCoreAPI,
			},
			args: args{
				req: &KeyInsertRequest{
					"mack",
					"0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309",
					"0x6246ddf254e0b4b4e7dffefc8adf69d212b98ac2b579c362b473fec8c40b4c0a",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := &AuthorModule{
				logger:     tt.fields.logger,
				coreAPI:    tt.fields.coreAPI,
				txStateAPI: tt.fields.txStateAPI,
			}
			if err := am.InsertKey(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("AuthorModule.InsertKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthorModule_HasKey(t *testing.T) {
	mockCoreAPITrue := &apimocks.CoreAPI{}
	mockCoreAPITrue.On("HasKey", mock.Anything, mock.Anything).Return(true, nil)

	mockCoreAPIFalse := &apimocks.CoreAPI{}
	mockCoreAPIFalse.On("HasKey", mock.Anything, mock.Anything).Return(false, nil)

	mockCoreAPIErr := &apimocks.CoreAPI{}
	mockCoreAPIErr.On("HasKey", mock.Anything, mock.Anything).Return(false, fmt.Errorf("some error"))

	kr, err := keystore.NewSr25519Keyring()

	if err != nil {
		panic(err)
	}

	type fields struct {
		logger     log.LeveledLogger
		coreAPI    CoreAPI
		txStateAPI TransactionStateAPI
	}
	type args struct {
		r   *http.Request
		req *[]string
		res *bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantRes bool
	}{
		{
			name: "HasKey true",
			fields: fields{
				coreAPI: mockCoreAPITrue,
			},
			args: args{
				req: &[]string{kr.Alice().Public().Hex(), "babe"},
				res: new(bool),
			},
			wantRes: true,
		},
		{
			name: "HasKey false",
			fields: fields{
				coreAPI: mockCoreAPIFalse,
			},
			args: args{
				req: &[]string{kr.Alice().Public().Hex(), "babe"},
				res: new(bool),
			},
			wantRes: false,
		},
		{
			name: "HasKey error",
			fields: fields{
				coreAPI: mockCoreAPIErr,
			},
			args: args{
				req: &[]string{kr.Alice().Public().Hex(), "babe"},
				res: new(bool),
			},
			wantRes: false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &AuthorModule{
				logger:     tt.fields.logger,
				coreAPI:    tt.fields.coreAPI,
				txStateAPI: tt.fields.txStateAPI,
			}
			if err := cm.HasKey(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("AuthorModule.HasKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.wantRes, *tt.args.res); diff != "" {
				t.Errorf("unexpected response: %s", diff)
			}
		})
	}
}
