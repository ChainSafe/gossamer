package parachaininteraction

import (
	"bytes"
	"errors"
	"fmt"

	runtimewasmer "github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrCodeEmpty         = errors.New("code is empty")
	ErrWASMDecompress    = errors.New("wasm decompression failed")
	ErrInstanceIsStopped = errors.New("instance is stopped")
)

func setupVM(code []byte) (*Instance, error) {
	cfg := runtimewasmer.Config{}

	instance, err := runtimewasmer.NewInstance(code, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}
	return &Instance{instance}, nil
}

type Instance struct {
	*runtimewasmer.Instance
}

func (in *Instance) ValidateBlock(params ValidationParameters) (
	*ValidationResult, error) {

	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(params)
	if err != nil {
		return nil, fmt.Errorf("encoding validation parameters: %w", err)
	}

	encodedValidationResult, err := in.Exec("validate_block", buffer.Bytes())
	if err != nil {
		return nil, err
	}

	validationResult := ValidationResult{}
	err = scale.Unmarshal(encodedValidationResult, &validationResult)
	if err != nil {
		return nil, fmt.Errorf("scale decoding: %w", err)
	}
	return &validationResult, nil
}
