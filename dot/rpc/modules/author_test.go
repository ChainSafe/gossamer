package modules

import (
	"fmt"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/transaction"
	log "github.com/ChainSafe/log15"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/mock"
)

func TestAuthorModule_SubmitExtrinsic(t *testing.T) {
	errMockCoreAPI := &apimocks.MockCoreAPI{}
	errMockCoreAPI.On("HandleSubmittedExtrinsic", mock.AnythingOfType("types.Extrinsic")).Return(fmt.Errorf("some error"))

	mockCoreAPI := &apimocks.MockCoreAPI{}
	mockCoreAPI.On("HandleSubmittedExtrinsic", mock.AnythingOfType("types.Extrinsic")).Return(nil)

	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	var testExt = common.MustHexToBytes("0x410284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01f8efbe48487e57a22abf7e3acd491b7f3528a33a111b1298601554863d27eb129eaa4e718e1365414ff3d028b62bebc651194c6b5001e5c2839b982757e08a8c0000000600ff8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a480b00c465f14670")

	// invalid transaction (above tx, with last byte changed)
	//nolint
	var testInvalidExt = []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 143}

	type fields struct {
		logger     log.Logger
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
				logger:  log.New("service", "RPC", "module", "author"),
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
				logger:  log.New("service", "RPC", "module", "author"),
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
		logger     log.Logger
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
				logger:     log.New("service", "RPC", "module", "author"),
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
				logger:     log.New("service", "RPC", "module", "author"),
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
	mockCoreAPI := &apimocks.MockCoreAPI{}
	mockCoreAPI.On("InsertKey", mock.Anything).Return(nil)

	type fields struct {
		logger     log.Logger
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
				logger:  log.New("service", "RPC", "module", "author"),
				coreAPI: mockCoreAPI,
			},
			args: args{
				req: &KeyInsertRequest{
					"babe",
					"0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309",
					"0x6246ddf254e0b4b4e7dffefc8adf69d212b98ac2b579c362b473fec8c40b4c0a",
				},
			},
		},
		{
			name: "happy path, gran keytype",
			fields: fields{
				logger:  log.New("service", "RPC", "module", "author"),
				coreAPI: mockCoreAPI,
			},
			args: args{
				req: &KeyInsertRequest{"gran",
					"0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309b7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309",
					"0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309",
				},
			},
		},
		{
			name: "invalid key",
			fields: fields{
				logger:  log.New("service", "RPC", "module", "author"),
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
				logger:  log.New("service", "RPC", "module", "author"),
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
	mockCoreAPITrue := &apimocks.MockCoreAPI{}
	mockCoreAPITrue.On("HasKey", mock.Anything, mock.Anything).Return(true, nil)

	mockCoreAPIFalse := &apimocks.MockCoreAPI{}
	mockCoreAPIFalse.On("HasKey", mock.Anything, mock.Anything).Return(false, nil)

	mockCoreAPIErr := &apimocks.MockCoreAPI{}
	mockCoreAPIErr.On("HasKey", mock.Anything, mock.Anything).Return(false, fmt.Errorf("some error"))

	kr, err := keystore.NewSr25519Keyring()

	if err != nil {
		panic(err)
	}

	type fields struct {
		logger     log.Logger
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
