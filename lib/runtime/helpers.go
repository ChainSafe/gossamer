package runtime

import (
	"errors"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/gorilla/rpc/v2/json2"
)

// ErrCannotValidateTx is returned if the call to runtime function TaggedTransactionQueueValidateTransaction fails
var ErrCannotValidateTx = errors.New("could not validate transaction")

// ErrInvalidTransaction is returned if the call to runtime function TaggedTransactionQueueValidateTransaction fails with
//  value of [1, 0, x]
var ErrInvalidTransaction = &json2.Error{Code: 1010, Message: "Invalid Transaction"}

// ErrUnknownTransaction is returned if the call to runtime function TaggedTransactionQueueValidateTransaction fails with
//  value of [1, 1, x]
var ErrUnknownTransaction = &json2.Error{Code: 1011, Message: "Unknown Transaction Validity"}

// ValidateTransaction runs the extrinsic through runtime function TaggedTransactionQueue_validate_transaction and returns *Validity
func (r *Runtime) ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error) {
	ret, err := r.Exec(TaggedTransactionQueueValidateTransaction, e)
	if err != nil {
		return nil, err
	}

	if ret[0] != 0 {
		return nil, determineError(ret)
	}

	v := transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	_, err = scale.Decode(ret[1:], v)

	return v, err
}

func determineError(res []byte) error {
	// confirm we have an error
	if res[0] == 0 {
		return nil
	}

	if res[1] == 0 {
		// transaction is invalid
		return ErrInvalidTransaction
	}

	if res[1] == 1 {
		// transaction validity can't be determined
		return ErrUnknownTransaction
	}

	return ErrCannotValidateTx
}

// BabeConfiguration gets the configuration data for BABE from the runtime
func (r *Runtime) BabeConfiguration() (*types.BabeConfiguration, error) {
	data, err := r.Exec(BabeAPIConfiguration, []byte{})
	if err != nil {
		return nil, err
	}

	bc := new(types.BabeConfiguration)
	_, err = scale.Decode(data, bc)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// GrandpaAuthorities returns the genesis authorities from the runtime
// TODO: this seems to be out-of-date, the call is now named Grandpa_authorities and takes a block number.
func (r *Runtime) GrandpaAuthorities() ([]*types.AuthorityData, error) {
	ret, err := r.Exec(AuraAPIAuthorities, []byte{})
	if err != nil {
		return nil, err
	}

	decodedKeys, err := scale.Decode(ret, [][32]byte{})
	if err != nil {
		return nil, err
	}

	keys := decodedKeys.([][32]byte)
	authsRaw := make([]*types.AuthorityDataRaw, len(keys))

	for i, key := range keys {
		authsRaw[i] = &types.AuthorityDataRaw{
			ID:     key,
			Weight: 1,
		}
	}

	auths := make([]*types.AuthorityData, len(keys))
	for i, auth := range authsRaw {
		auths[i] = new(types.AuthorityData)
		err = auths[i].FromRaw(auth)
		if err != nil {
			return nil, err
		}
	}

	return auths, err
}

// InitializeBlock calls runtime API function Core_initialize_block
func (r *Runtime) InitializeBlock(blockHeader []byte) error {
	_, err := r.Exec(CoreInitializeBlock, blockHeader)
	return err
}

// InherentExtrinsics calls runtime API function BlockBuilder_inherent_extrinsics
func (r *Runtime) InherentExtrinsics(data []byte) ([]byte, error) {
	return r.Exec(BlockBuilderInherentExtrinsics, data)
}

// ApplyExtrinsic calls runtime API function BlockBuilder_apply_extrinsic
func (r *Runtime) ApplyExtrinsic(data types.Extrinsic) ([]byte, error) {
	return r.Exec(BlockBuilderApplyExtrinsic, data)
}

// FinalizeBlock calls runtime API function BlockBuilder_finalize_block
func (r *Runtime) FinalizeBlock() (*types.Header, error) {
	data, err := r.Exec(BlockBuilderFinalizeBlock, []byte{})
	if err != nil {
		return nil, err
	}

	bh := new(types.Header)
	_, err = scale.Decode(data, bh)
	if err != nil {
		return nil, err
	}

	return bh, nil
}
