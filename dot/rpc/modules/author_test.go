// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorModule_HasSessionKeys(t *testing.T) {
	const testReq = "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d34309a9d2a24213896ff06895db16" +
		"aade8b6502f3a71cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026343" +
		"09a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026"
	pkeys := common.MustHexToBytes("0x0102d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d3430" +
		"9a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71" +
		"cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026")
	data := common.MustHexToBytes("0x011080d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a" +
		"56da27d6772616e8034309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc3852042602662616265803430" +
		"9a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026696d6f6e8034309a9d2a24213896ff06895" +
		"db16aade8b6502f3a71cf56374cc3852042602661756469")

	ctrl := gomock.NewController(t)

	coreMockAPIUnmarshalErr := mocks.NewMockCoreAPI(ctrl)
	coreMockAPIUnmarshalErr.EXPECT().DecodeSessionKeys([]byte{0x4, 0x1}).Return([]byte{0x4, 0x1}, nil)

	coreMockAPIOk := mocks.NewMockCoreAPI(ctrl)
	coreMockAPIOk.EXPECT().DecodeSessionKeys(pkeys).Return(data, nil)
	coreMockAPIOk.EXPECT().HasKey(gomock.Any(), gomock.Any()).
		Return(true, nil).Times(4)

	coreMockAPIErr := mocks.NewMockCoreAPI(ctrl)
	coreMockAPIErr.EXPECT().DecodeSessionKeys(pkeys).Return(data, nil)
	coreMockAPIErr.EXPECT().HasKey(gomock.Any(),
		gomock.Any()).
		Return(false, errors.New("HasKey err"))

	coreMockAPIInvalidDec := mocks.NewMockCoreAPI(ctrl)
	coreMockAPIInvalidDec.EXPECT().DecodeSessionKeys(pkeys).Return([]byte{0x0}, nil)

	type fields struct {
		authorModule *AuthorModule
	}
	type args struct {
		r   *http.Request
		req *HasSessionKeyRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    HasSessionKeyResponse
	}{
		{
			name: "Empty_Request",
			fields: fields{
				authorModule: NewAuthorModule(log.New(log.SetWriter(io.Discard)), nil, nil),
			},
			args: args{
				req: &HasSessionKeyRequest{},
			},
			exp:    false,
			expErr: errors.New("could not byteify non 0x prefixed string: "),
		},
		{
			name: "decodeSessionKeys_err",
			fields: fields{
				authorModule: NewAuthorModule(log.New(log.SetWriter(io.Discard)), coreMockAPIUnmarshalErr, nil),
			},
			args: args{
				req: &HasSessionKeyRequest{"0x01"},
			},
			exp:    false,
			expErr: errors.New("unsupported option: value: 4, bytes: [1]"),
		},
		{
			name: "happy_path",
			fields: fields{
				authorModule: NewAuthorModule(log.New(log.SetWriter(io.Discard)), coreMockAPIOk, nil),
			},
			args: args{
				req: &HasSessionKeyRequest{testReq},
			},
			exp: true,
		},
		{
			name: "doesnt_have_key",
			fields: fields{
				authorModule: NewAuthorModule(log.New(log.SetWriter(io.Discard)), coreMockAPIErr, nil),
			},
			args: args{
				req: &HasSessionKeyRequest{testReq},
			},
			exp:    false,
			expErr: errors.New("HasKey err"),
		},
		{
			name: "Empty_decodedKeys",
			fields: fields{
				authorModule: NewAuthorModule(log.New(log.SetWriter(io.Discard)), coreMockAPIInvalidDec, nil),
			},
			args: args{
				req: &HasSessionKeyRequest{testReq},
			},
			exp: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := tt.fields.authorModule
			var res HasSessionKeyResponse
			err := am.HasSessionKeys(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestAuthorModule_SubmitExtrinsic(t *testing.T) {
	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	var testExt = common.MustHexToBytes("0x410284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27" +
		"d01f8efbe48487e57a22abf7e3acd491b7f3528a33a111b1298601554863d27eb129eaa4e718e1365414ff3d028b62bebc651194c6b" +
		"5001e5c2839b982757e08a8c0000000600ff8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a480b00c465f14670")
	// invalid transaction (above tx, with last byte changed)
	var testInvalidExt = []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130,
		44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115,
		99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242,
		106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137,
		201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163,
		149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245,
		24, 225, 37, 154, 163, 143}

	ctrl := gomock.NewController(t)

	errMockCoreAPI := mocks.NewMockCoreAPI(ctrl)
	errMockCoreAPI.EXPECT().HandleSubmittedExtrinsic(
		types.Extrinsic(common.MustHexToBytes(fmt.Sprintf("0x%x", testInvalidExt)))).Return(fmt.Errorf("some error"))

	mockCoreAPI := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPI.EXPECT().HandleSubmittedExtrinsic(
		types.Extrinsic(common.MustHexToBytes(fmt.Sprintf("0x%x", testExt)))).Return(nil)
	type fields struct {
		logger     Infoer
		coreAPI    CoreAPI
		txStateAPI TransactionStateAPI
	}
	type args struct {
		r   *http.Request
		req *Extrinsic
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		expErr  error
		wantRes ExtrinsicHashResponse
	}{
		{
			name: "HexToBytes_error",
			args: args{
				req: &Extrinsic{fmt.Sprintf("%x", "1")},
			},
			expErr:  fmt.Errorf("could not byteify non 0x prefixed string: 31"),
			wantRes: ExtrinsicHashResponse(""),
		},
		{
			name: "HandleSubmittedExtrinsic_error",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: errMockCoreAPI,
			},
			args: args{
				req: &Extrinsic{fmt.Sprintf("0x%x", testInvalidExt)},
			},
			expErr:  fmt.Errorf("some error"),
			wantRes: ExtrinsicHashResponse(types.Extrinsic(testInvalidExt).Hash().String()),
		},
		{
			name: "happy_path",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: mockCoreAPI,
			},
			args: args{
				req: &Extrinsic{fmt.Sprintf("0x%x", testExt)},
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
			res := ExtrinsicHashResponse("")
			err := am.SubmitExtrinsic(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantRes, res)
		})
	}
}

func TestAuthorModule_PendingExtrinsics(t *testing.T) {
	ctrl := gomock.NewController(t)

	emptyMockTransactionStateAPI := mocks.NewMockTransactionStateAPI(ctrl)
	emptyMockTransactionStateAPI.EXPECT().Pending().Return([]*transaction.ValidTransaction{})

	mockTransactionStateAPI := mocks.NewMockTransactionStateAPI(ctrl)
	mockTransactionStateAPI.EXPECT().Pending().Return([]*transaction.ValidTransaction{
		{
			Extrinsic: types.NewExtrinsic([]byte("someExtrinsic")),
		},
		{
			Extrinsic: types.NewExtrinsic([]byte("someExtrinsic1")),
		},
	})

	type fields struct {
		logger     Infoer
		coreAPI    CoreAPI
		txStateAPI TransactionStateAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		expErr  error
		wantRes PendingExtrinsicsResponse
	}{
		{
			name: "no_pending",
			fields: fields{
				logger:     log.New(log.SetWriter(io.Discard)),
				txStateAPI: emptyMockTransactionStateAPI,
			},
			wantRes: PendingExtrinsicsResponse{},
		},
		{
			name: "two_pending",
			fields: fields{
				logger:     log.New(log.SetWriter(io.Discard)),
				txStateAPI: mockTransactionStateAPI,
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
			res := PendingExtrinsicsResponse{}
			err := cm.PendingExtrinsics(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantRes, res)
		})
	}
}

func TestAuthorModule_InsertKey(t *testing.T) {
	kp1, err := sr25519.NewKeypairFromSeed(
		common.MustHexToBytes("0x6246ddf254e0b4b4e7dffefc8adf69d212b98ac2b579c362b473fec8c40b4c0a"))
	require.NoError(t, err)
	// this is needed to set the internal field *schnorrkel.PublicKey.compressedKey which causes DeepEqual to fail
	_ = kp1.Public().Hex()

	kp2, err := ed25519.NewKeypairFromSeed(
		common.MustHexToBytes("0xb48004c6e1625282313b07d1c9950935e86894a2e4f21fb1ffee9854d180c781"))
	require.NoError(t, err)
	_ = kp2.Public().Hex()

	kp3, err := sr25519.NewKeypairFromSeed(
		common.MustHexToBytes("0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309"))
	require.NoError(t, err)
	_ = kp3.Public().Hex()

	ctrl := gomock.NewController(t)

	mockCoreAPIHappyBabe := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIHappyBabe.EXPECT().InsertKey(kp1, "babe").Return(nil)

	mockCoreAPIHappyGran := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIHappyGran.EXPECT().InsertKey(kp2, "gran").Return(nil)

	type fields struct {
		logger     Infoer
		coreAPI    CoreAPI
		txStateAPI TransactionStateAPI
	}
	type args struct {
		r   *http.Request
		req *KeyInsertRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
	}{
		{
			name: "happy_path",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: mockCoreAPIHappyBabe,
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
			name: "happy_path,_gran_keytype",
			fields: fields{
				logger:  log.New(log.SetWriter(io.Discard)),
				coreAPI: mockCoreAPIHappyGran,
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
			name: "invalid_key",
			fields: fields{
				logger: log.New(log.SetWriter(io.Discard)),
			},
			args: args{
				req: &KeyInsertRequest{"babe",
					"0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309",
					"0x0000000000000000000000000000000000000000000000000000000000000000",
				},
			},
			expErr: ErrProvidedKeyDoesNotMatch,
		},
		{
			name: "unknown_key",
			fields: fields{
				logger: log.New(log.SetWriter(io.Discard)),
			},
			args: args{
				req: &KeyInsertRequest{
					"mack",
					"0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309",
					"0x6246ddf254e0b4b4e7dffefc8adf69d212b98ac2b579c362b473fec8c40b4c0a",
				},
			},
			expErr: errors.New("cannot decode key: invalid key type"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := &AuthorModule{
				logger:     tt.fields.logger,
				coreAPI:    tt.fields.coreAPI,
				txStateAPI: tt.fields.txStateAPI,
			}
			var res KeyInsertResponse
			err := am.InsertKey(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthorModule_HasKey(t *testing.T) {
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)

	mockCoreAPITrue := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPITrue.EXPECT().HasKey(kr.Alice().Public().Hex(), "babe").Return(true, nil)

	mockCoreAPIFalse := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIFalse.EXPECT().HasKey(kr.Alice().Public().Hex(), "babe").Return(false, nil)

	mockCoreAPIErr := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIErr.EXPECT().HasKey(kr.Alice().Public().Hex(), "babe").Return(false, fmt.Errorf("some error"))

	type fields struct {
		logger     Infoer
		coreAPI    CoreAPI
		txStateAPI TransactionStateAPI
	}
	type args struct {
		r   *http.Request
		req *[]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		expErr  error
		wantRes bool
	}{
		{
			name: "HasKey_true",
			fields: fields{
				coreAPI: mockCoreAPITrue,
			},
			args: args{
				req: &[]string{kr.Alice().Public().Hex(), "babe"},
			},
			wantRes: true,
		},
		{
			name: "HasKey_false",
			fields: fields{
				coreAPI: mockCoreAPIFalse,
			},
			args: args{
				req: &[]string{kr.Alice().Public().Hex(), "babe"},
			},
			wantRes: false,
		},
		{
			name: "HasKey_error",
			fields: fields{
				coreAPI: mockCoreAPIErr,
			},
			args: args{
				req: &[]string{kr.Alice().Public().Hex(), "babe"},
			},
			wantRes: false,
			expErr:  fmt.Errorf("some error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &AuthorModule{
				logger:     tt.fields.logger,
				coreAPI:    tt.fields.coreAPI,
				txStateAPI: tt.fields.txStateAPI,
			}
			res := false
			err := cm.HasKey(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantRes, res)
		})
	}
}
