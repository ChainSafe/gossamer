// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"bytes"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// Version represents the data returned by runtime call core_version
type Version interface {
	SpecName() []byte
	ImplName() []byte
	AuthoringVersion() uint32
	SpecVersion() uint32
	ImplVersion() uint32
	APIItems() []*APIItem
	TransactionVersion() uint32
	Encode() ([]byte, error)
}

// APIItem struct to hold runtime API Name and Version
type APIItem struct {
	Name [8]byte
	Ver  uint32
}

// LegacyVersionData is the runtime version info returned by legacy runtimes
type LegacyVersionData struct {
	specName         []byte
	implName         []byte
	authoringVersion uint32
	specVersion      uint32
	implVersion      uint32
	apiItems         []*APIItem
}

// NewLegacyVersionData returns a new LegacyVersionData
func NewLegacyVersionData(specName, implName []byte, authoringVersion, specVersion, implVersion uint32, apiItems []*APIItem) *LegacyVersionData {
	return &LegacyVersionData{
		specName:         specName,
		implName:         implName,
		authoringVersion: authoringVersion,
		specVersion:      specVersion,
		implVersion:      implVersion,
		apiItems:         apiItems,
	}
}

// SpecName returns the spec name
func (v *LegacyVersionData) SpecName() []byte {
	return v.specName
}

// ImplName returns the implementation name
func (v *LegacyVersionData) ImplName() []byte {
	return v.implName
}

// AuthoringVersion returns the authoring version
func (v *LegacyVersionData) AuthoringVersion() uint32 {
	return v.authoringVersion
}

// SpecVersion returns the spec version
func (v *LegacyVersionData) SpecVersion() uint32 {
	return v.specVersion
}

// ImplVersion returns the implementation version
func (v *LegacyVersionData) ImplVersion() uint32 {
	return v.implVersion
}

// APIItems returns the API items
func (v *LegacyVersionData) APIItems() []*APIItem {
	return v.apiItems
}

// TransactionVersion returns the transaction version
func (v *LegacyVersionData) TransactionVersion() uint32 {
	return 0
}

// Encode returns the SCALE encoding of the Version
func (v *LegacyVersionData) Encode() ([]byte, error) {
	info := &struct {
		SpecName         []byte
		ImplName         []byte
		AuthoringVersion uint32
		SpecVersion      uint32
		ImplVersion      uint32
	}{
		SpecName:         v.specName,
		ImplName:         v.implName,
		AuthoringVersion: v.authoringVersion,
		SpecVersion:      v.specVersion,
		ImplVersion:      v.implVersion,
	}

	enc, err := scale.Encode(info)
	if err != nil {
		return nil, err
	}

	b, err := scale.Encode(big.NewInt(int64(len(v.apiItems))))
	if err != nil {
		return nil, err
	}
	enc = append(enc, b...)

	for _, apiItem := range v.apiItems {
		enc = append(enc, apiItem.Name[:]...)

		b, err = scale.Encode(apiItem.Ver)
		if err != nil {
			return nil, err
		}
		enc = append(enc, b...)
	}

	return enc, nil
}

// Decode to scale decode []byte to VersionAPI struct
func (v *LegacyVersionData) Decode(in []byte) error {
	r := &bytes.Buffer{}
	_, err := r.Write(in)
	if err != nil {
		return err
	}
	sd := scale.Decoder{Reader: r}

	type Info struct {
		SpecName         []byte
		ImplName         []byte
		AuthoringVersion uint32
		SpecVersion      uint32
		ImplVersion      uint32
	}

	ret, err := sd.Decode(new(Info))
	if err != nil {
		return err
	}

	info := ret.(*Info)

	v.specName = info.SpecName
	v.implName = info.ImplName
	v.authoringVersion = info.AuthoringVersion
	v.specVersion = info.SpecVersion
	v.implVersion = info.ImplVersion

	numApis, err := sd.DecodeInteger()
	if err != nil {
		return err
	}

	for i := 0; i < int(numApis); i++ {
		name, err := common.Read8Bytes(r) //nolint
		if err != nil {
			return err
		}

		version, err := common.ReadUint32(r)
		if err != nil {
			return err
		}

		v.apiItems = append(v.apiItems, &APIItem{
			Name: name,
			Ver:  version,
		})
	}

	return nil
}

// VersionData is the runtime version info returned by v0.8 runtimes
type VersionData struct {
	specName           []byte
	implName           []byte
	authoringVersion   uint32
	specVersion        uint32
	implVersion        uint32
	apiItems           []*APIItem
	transactionVersion uint32
}

// NewVersionData returns a new VersionData
func NewVersionData(specName, implName []byte, authoringVersion, specVersion, implVersion uint32, apiItems []*APIItem, transactionVersion uint32) *VersionData {
	return &VersionData{
		specName:           specName,
		implName:           implName,
		authoringVersion:   authoringVersion,
		specVersion:        specVersion,
		implVersion:        implVersion,
		apiItems:           apiItems,
		transactionVersion: transactionVersion,
	}
}

// SpecName returns the spec name
func (v *VersionData) SpecName() []byte {
	return v.specName
}

// ImplName returns the implementation name
func (v *VersionData) ImplName() []byte {
	return v.implName
}

// AuthoringVersion returns the authoring version
func (v *VersionData) AuthoringVersion() uint32 {
	return v.authoringVersion
}

// SpecVersion returns the spec version
func (v *VersionData) SpecVersion() uint32 {
	return v.specVersion
}

// ImplVersion returns the implementation version
func (v *VersionData) ImplVersion() uint32 {
	return v.implVersion
}

// APIItems returns the API items
func (v *VersionData) APIItems() []*APIItem {
	return v.apiItems
}

// TransactionVersion returns the transaction version
func (v *VersionData) TransactionVersion() uint32 {
	return v.transactionVersion
}

// Encode returns the SCALE encoding of the Version
func (v *VersionData) Encode() ([]byte, error) {
	info := &struct {
		SpecName         []byte
		ImplName         []byte
		AuthoringVersion uint32
		SpecVersion      uint32
		ImplVersion      uint32
	}{
		SpecName:         v.specName,
		ImplName:         v.implName,
		AuthoringVersion: v.authoringVersion,
		SpecVersion:      v.specVersion,
		ImplVersion:      v.implVersion,
	}

	enc, err := scale.Encode(info)
	if err != nil {
		return nil, err
	}

	b, err := scale.Encode(big.NewInt(int64(len(v.apiItems))))
	if err != nil {
		return nil, err
	}
	enc = append(enc, b...)

	for _, apiItem := range v.apiItems {
		enc = append(enc, apiItem.Name[:]...)

		b, err = scale.Encode(apiItem.Ver)
		if err != nil {
			return nil, err
		}
		enc = append(enc, b...)
	}

	b, err = scale.Encode(v.transactionVersion)
	if err != nil {
		return nil, err
	}
	enc = append(enc, b...)

	return enc, nil
}

// Decode to scale decode []byte to VersionAPI struct
func (v *VersionData) Decode(in []byte) error {
	r := &bytes.Buffer{}
	_, err := r.Write(in)
	if err != nil {
		return err
	}
	sd := scale.Decoder{Reader: r}

	type Info struct {
		SpecName         []byte
		ImplName         []byte
		AuthoringVersion uint32
		SpecVersion      uint32
		ImplVersion      uint32
	}

	ret, err := sd.Decode(new(Info))
	if err != nil {
		return err
	}

	info := ret.(*Info)

	v.specName = info.SpecName
	v.implName = info.ImplName
	v.authoringVersion = info.AuthoringVersion
	v.specVersion = info.SpecVersion
	v.implVersion = info.ImplVersion

	numApis, err := sd.DecodeInteger()
	if err != nil {
		return err
	}

	for i := 0; i < int(numApis); i++ {
		name, err := common.Read8Bytes(r) //nolint
		if err != nil {
			return err
		}

		version, err := common.ReadUint32(r)
		if err != nil {
			return err
		}

		v.apiItems = append(v.apiItems, &APIItem{
			Name: name,
			Ver:  version,
		})
	}

	v.transactionVersion, err = common.ReadUint32(r)
	return err
}
