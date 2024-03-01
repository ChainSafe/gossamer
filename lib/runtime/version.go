// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// APIItem struct to hold runtime API Name and Version
type APIItem struct {
	Name [8]byte
	Ver  uint32
}

// Version is the runtime version info.
type Version struct {
	SpecName           []byte
	ImplName           []byte
	AuthoringVersion   uint32
	SpecVersion        uint32
	ImplVersion        uint32
	APIItems           []APIItem
	TransactionVersion uint32
	StateVersion       uint8
}

var (
	ErrDecodingVersionField = errors.New("decoding version field")
)

// TaggedTransactionQueueVersion returns the TaggedTransactionQueue API version
func (v Version) TaggedTransactionQueueVersion() (txQueueVersion uint32, err error) {
	encodedTaggedTransactionQueue, err := common.Blake2b8([]byte("TaggedTransactionQueue"))
	if err != nil {
		return 0, fmt.Errorf("getting blake2b8: %s", err)
	}
	for _, apiItem := range v.APIItems {
		if apiItem.Name == encodedTaggedTransactionQueue {
			return apiItem.Ver, nil
		}
	}
	return 0, errors.New("taggedTransactionQueueAPI not found")
}

// DecodeVersion scale decodes the encoded version data.
// For older version data with missing fields (such as `transaction_version`)
// the missing field is set to its zero value (such as `0`).
func DecodeVersion(encoded []byte) (version Version, err error) {
	reader := bytes.NewReader(encoded)
	decoder := scale.NewDecoder(reader)

	type namedValue struct {
		// name is the field name used to wrap eventual codec errors
		name string
		// value is the field value to handle
		value interface{}
	}

	requiredFields := [...]namedValue{
		{name: "spec name", value: &version.SpecName},
		{name: "impl name", value: &version.ImplName},
		{name: "authoring version", value: &version.AuthoringVersion},
		{name: "spec version", value: &version.SpecVersion},
		{name: "impl version", value: &version.ImplVersion},
		{name: "API items", value: &version.APIItems},
	}
	for _, requiredField := range requiredFields {
		err = decoder.Decode(requiredField.value)
		if err != nil {
			return Version{}, fmt.Errorf("%w %s: %s", ErrDecodingVersionField, requiredField.name, err)
		}
	}

	optionalFields := [...]namedValue{
		{name: "transaction version", value: &version.TransactionVersion},
		{name: "state version", value: &version.StateVersion},
	}
	for _, optionalField := range optionalFields {
		err = decoder.Decode(optionalField.value)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return version, nil
			}
			return Version{}, fmt.Errorf("%w %s: %s", ErrDecodingVersionField, optionalField.name, err)
		}
	}

	return version, nil
}
