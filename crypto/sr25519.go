package crypto

import (
	"errors"
	"fmt"
	"sync"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

const (
	SR25519_CHAINCODE_SIZE  = 32
	SR25519_KEYPAIR_SIZE    = 96
	SR25519_PUBLIC_SIZE     = 32
	SR25519_SECRET_SIZE     = 64
	SR25519_SEED_SIZE       = 32
	SR25519_SIGNATURE_SIZE  = 64
	SR25519_VRF_OUTPUT_SIZE = 32
	SR25519_VRF_PROOF_SIZE  = 64
)

type SchnorrkelExecutor struct {
	vm   wasm.Instance
	lock sync.Mutex
}

func NewSchnorrkelExecutor(fp string) (*SchnorrkelExecutor, error) {
	// Reads the WebAssembly module as bytes.
	bytes, err := wasm.ReadBytes(fp)
	if err != nil {
		return nil, fmt.Errorf("cannot read bytes: %s", err)
	}

	// Instantiates the WebAssembly module.
	instance, err := wasm.NewInstance(bytes)
	if err != nil {
		return nil, err
	}

	return &SchnorrkelExecutor{
		vm:   instance,
	}, nil
}

func (se *SchnorrkelExecutor) Stop() {
	se.vm.Close()
}

func (se *SchnorrkelExecutor) Sr25519KeypairFromSeed(seed []byte) ([]byte, error) {
	var out_ptr int32 = 1
	var seed_ptr int32 = out_ptr + SR25519_KEYPAIR_SIZE

	se.lock.Lock()
	defer se.lock.Unlock()

	mem := se.vm.Memory.Data()
	copy(mem[seed_ptr:seed_ptr+SR25519_SEED_SIZE], seed)

	_, err := se.Exec("sr25519_keypair_from_seed", out_ptr, seed_ptr)
	if err != nil {
		return nil, err
	}

	keypair_out := make([]byte, SR25519_KEYPAIR_SIZE)
	copy(keypair_out, mem[out_ptr:out_ptr+SR25519_KEYPAIR_SIZE])
	return keypair_out, nil
}

func (se *SchnorrkelExecutor) Sr25519DeriveKeypairHard(keypair, chaincode []byte) ([]byte, error) {
	var out_ptr int32 = 1
	var pair_ptr int32 = out_ptr + SR25519_KEYPAIR_SIZE
	var cc_ptr int32 = pair_ptr + SR25519_KEYPAIR_SIZE

	se.lock.Lock()
	defer se.lock.Unlock()

	mem := se.vm.Memory.Data()
	copy(mem[pair_ptr:pair_ptr+SR25519_KEYPAIR_SIZE], keypair)
	copy(mem[cc_ptr:cc_ptr+SR25519_CHAINCODE_SIZE], chaincode)

	_, err := se.Exec("sr25519_derive_keypair_hard", out_ptr, pair_ptr, cc_ptr)
	if err != nil {
		return nil, err
	}

	keypair_out := make([]byte, SR25519_KEYPAIR_SIZE)
	copy(keypair_out, mem[out_ptr:out_ptr+SR25519_KEYPAIR_SIZE])
	return keypair_out, nil
}

func (se *SchnorrkelExecutor) Sr25519DeriveKeypairSoft(keypair, chaincode []byte) ([]byte, error) {
	var out_ptr int32 = 1
	var pair_ptr int32 = out_ptr + SR25519_KEYPAIR_SIZE
	var cc_ptr int32 = pair_ptr + SR25519_KEYPAIR_SIZE

	se.lock.Lock()
	defer se.lock.Unlock()

	mem := se.vm.Memory.Data()
	copy(mem[pair_ptr:pair_ptr+SR25519_KEYPAIR_SIZE], keypair)
	copy(mem[cc_ptr:cc_ptr+SR25519_CHAINCODE_SIZE], chaincode)

	_, err := se.Exec("sr25519_derive_keypair_soft", out_ptr, pair_ptr, cc_ptr)
	if err != nil {
		return nil, err
	}

	keypair_out := make([]byte, SR25519_KEYPAIR_SIZE)
	copy(keypair_out, mem[out_ptr:out_ptr+SR25519_KEYPAIR_SIZE])
	return keypair_out, nil
}

func (se *SchnorrkelExecutor) Sr25519DerivePublicSoft(pubkey, chaincode []byte) ([]byte, error) {
	var pubkey_out_ptr int32 = 1
	var public_ptr int32 = pubkey_out_ptr + SR25519_PUBLIC_SIZE
	var cc_ptr int32 = public_ptr + SR25519_PUBLIC_SIZE

	se.lock.Lock()
	defer se.lock.Unlock()

	mem := se.vm.Memory.Data()
	copy(mem[public_ptr:public_ptr+SR25519_PUBLIC_SIZE], pubkey)
	copy(mem[cc_ptr:cc_ptr+SR25519_CHAINCODE_SIZE], chaincode)

	_, err := se.Exec("sr25519_derive_public_soft", pubkey_out_ptr, public_ptr, cc_ptr)
	if err != nil {
		return nil, err
	}

	pubkey_out := make([]byte, SR25519_PUBLIC_SIZE)
	copy(pubkey_out, mem[pubkey_out_ptr:pubkey_out_ptr+SR25519_PUBLIC_SIZE])
	return pubkey_out, nil
}

func (se *SchnorrkelExecutor) Sr25519Sign(public, secret, message []byte) ([]byte, error) {
	signature_out_ptr := 1
	public_ptr := signature_out_ptr + SR25519_SIGNATURE_SIZE
	secret_ptr := public_ptr + SR25519_PUBLIC_SIZE
	message_ptr := secret_ptr + SR25519_SECRET_SIZE

	se.lock.Lock()
	defer se.lock.Unlock()

	mem := se.vm.Memory.Data()
	copy(mem[public_ptr:public_ptr+SR25519_PUBLIC_SIZE], public)
	copy(mem[secret_ptr:secret_ptr+SR25519_SECRET_SIZE], secret)
	copy(mem[message_ptr:message_ptr+len(message)], message)

	_, err := se.Exec("sr25519_sign", signature_out_ptr, public_ptr, secret_ptr, message_ptr, len(message))
	if err != nil {
		return nil, err
	}

	signature_out := make([]byte, SR25519_SIGNATURE_SIZE)
	copy(signature_out, mem[signature_out_ptr:signature_out_ptr+SR25519_SIGNATURE_SIZE])
	return signature_out, nil
}

func (se *SchnorrkelExecutor) Sr25519Verify(signature, message, pubkey []byte) (bool, error) {
	public_ptr := 1
	signature_ptr := public_ptr + SR25519_SECRET_SIZE
	message_ptr := signature_ptr + SR25519_SIGNATURE_SIZE	

	se.lock.Lock()
	defer se.lock.Unlock()

	mem := se.vm.Memory.Data()
	copy(mem[public_ptr:public_ptr+SR25519_PUBLIC_SIZE], pubkey)
	copy(mem[signature_ptr:signature_ptr+SR25519_SIGNATURE_SIZE], signature)
	copy(mem[message_ptr:message_ptr+len(message)], message)

	ret, err := se.Exec("sr25519_verify", signature_ptr, message_ptr, int32(len(message)), public_ptr)
	if err != nil {
		return false, err
	}

	return ret != 0, nil
}

// Returns output + proof of signature, and bool which says whether VRF random number was under limit or not
func (se *SchnorrkelExecutor) Sr25519VrfSign(keypair, message, limit []byte) ([]byte, bool, error) {
	out_and_proof_ptr := 1
	keypair_ptr := out_and_proof_ptr + SR25519_VRF_OUTPUT_SIZE + SR25519_VRF_PROOF_SIZE 
	message_ptr := keypair_ptr + SR25519_KEYPAIR_SIZE
	limit_ptr := message_ptr + len(message)

	se.lock.Lock()
	defer se.lock.Unlock()

	mem := se.vm.Memory.Data()
	copy(mem[keypair_ptr:keypair_ptr+SR25519_KEYPAIR_SIZE], keypair)
	copy(mem[message_ptr:message_ptr+len(message)], message)
	copy(mem[limit_ptr:limit_ptr+SR25519_VRF_OUTPUT_SIZE], limit)

	under_limit, err := se.Exec("sr25519_vrf_sign_if_less", out_and_proof_ptr, keypair_ptr, message_ptr, int32(len(message)), limit_ptr)
	if err != nil { 
		return nil, false, err
	}

	out_and_proof := make([]byte, SR25519_VRF_OUTPUT_SIZE+SR25519_VRF_PROOF_SIZE)
	copy(out_and_proof, mem[out_and_proof_ptr:out_and_proof_ptr+SR25519_VRF_OUTPUT_SIZE+SR25519_VRF_PROOF_SIZE])
;
	return out_and_proof, under_limit != 0, nil
}

func (se *SchnorrkelExecutor) Sr25519VrfVerify(public, message, out, proof []byte) (int64, error) {
	public_ptr := 1
	message_ptr := public_ptr + SR25519_PUBLIC_SIZE
	out_ptr := message_ptr + len(message)	
	proof_ptr := out_ptr + SR25519_VRF_OUTPUT_SIZE

	se.lock.Lock()
	defer se.lock.Unlock()

	mem := se.vm.Memory.Data()
	copy(mem[public_ptr:public_ptr+SR25519_PUBLIC_SIZE], public)
	copy(mem[message_ptr:message_ptr+len(message)], message)
	copy(mem[out_ptr:out_ptr+SR25519_VRF_OUTPUT_SIZE], out)
	copy(mem[proof_ptr:proof_ptr+SR25519_VRF_PROOF_SIZE], proof)

	ret, err := se.Exec("sr25519_vrf_verify", public_ptr, message_ptr, int32(len(message)), out_ptr, proof_ptr)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

func (se *SchnorrkelExecutor) Exec(function string, params... interface{}) (int64, error) {
	wasmFunc, ok := se.vm.Exports[function]
	if !ok {
		return 0, errors.New("could not find exported function")
	}

	res, err := wasmFunc(params...)
	if err != nil {
		return 0, err
	}

	resi := res.ToI64()
	return resi, nil
}