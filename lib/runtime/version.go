// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
)

//go:generate mockery --name Version --structname Version --case underscore --keeptree

// Version represents the data returned by runtime call core_version
type Version interface {
	SpecName() []byte
	ImplName() []byte
	AuthoringVersion() uint32
	SpecVersion() uint32
	ImplVersion() uint32
	APIItems() []APIItem
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
	apiItems         []APIItem
}

// NewLegacyVersionData returns a new LegacyVersionData
func NewLegacyVersionData(specName, implName []byte,
	authoringVersion, specVersion, implVersion uint32,
	apiItems []APIItem) *LegacyVersionData {
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
func (lvd *LegacyVersionData) SpecName() []byte {
	return lvd.specName
}

// ImplName returns the implementation name
func (lvd *LegacyVersionData) ImplName() []byte {
	return lvd.implName
}

// AuthoringVersion returns the authoring version
func (lvd *LegacyVersionData) AuthoringVersion() uint32 {
	return lvd.authoringVersion
}

// SpecVersion returns the spec version
func (lvd *LegacyVersionData) SpecVersion() uint32 {
	return lvd.specVersion
}

// ImplVersion returns the implementation version
func (lvd *LegacyVersionData) ImplVersion() uint32 {
	return lvd.implVersion
}

// APIItems returns the API items
func (lvd *LegacyVersionData) APIItems() []APIItem {
	return lvd.apiItems
}

// TransactionVersion returns the transaction version
func (lvd *LegacyVersionData) TransactionVersion() uint32 {
	return 0
}

type legacyVersionData struct {
	SpecName         []byte
	ImplName         []byte
	AuthoringVersion uint32
	SpecVersion      uint32
	ImplVersion      uint32
	APIItems         []APIItem
}

// Encode returns the SCALE encoding of the Version
func (lvd *LegacyVersionData) Encode() ([]byte, error) {
	info := legacyVersionData{
		SpecName:         lvd.specName,
		ImplName:         lvd.implName,
		AuthoringVersion: lvd.authoringVersion,
		SpecVersion:      lvd.specVersion,
		ImplVersion:      lvd.implVersion,
		APIItems:         lvd.apiItems,
	}

	enc, err := scale.Marshal(info)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

// Decode to scale decode []byte to VersionAPI struct
func (lvd *LegacyVersionData) Decode(in []byte) error {
	var info legacyVersionData
	err := scale.Unmarshal(in, &info)
	if err != nil {
		return err
	}

	lvd.specName = info.SpecName
	lvd.implName = info.ImplName
	lvd.authoringVersion = info.AuthoringVersion
	lvd.specVersion = info.SpecVersion
	lvd.implVersion = info.ImplVersion
	lvd.apiItems = info.APIItems

	return nil
}

// VersionData is the runtime version info returned by v0.8 runtimes
type VersionData struct {
	specName           []byte
	implName           []byte
	authoringVersion   uint32
	specVersion        uint32
	implVersion        uint32
	apiItems           []APIItem
	transactionVersion uint32
}

// NewVersionData returns a new VersionData
func NewVersionData(specName, implName []byte,
	authoringVersion, specVersion, implVersion uint32,
	apiItems []APIItem, transactionVersion uint32) *VersionData {
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
func (vd *VersionData) SpecName() []byte {
	return vd.specName
}

// ImplName returns the implementation name
func (vd *VersionData) ImplName() []byte {
	return vd.implName
}

// AuthoringVersion returns the authoring version
func (vd *VersionData) AuthoringVersion() uint32 {
	return vd.authoringVersion
}

// SpecVersion returns the spec version
func (vd *VersionData) SpecVersion() uint32 {
	return vd.specVersion
}

// ImplVersion returns the implementation version
func (vd *VersionData) ImplVersion() uint32 {
	return vd.implVersion
}

// APIItems returns the API items
func (vd *VersionData) APIItems() []APIItem {
	return vd.apiItems
}

// TransactionVersion returns the transaction version
func (vd *VersionData) TransactionVersion() uint32 {
	return vd.transactionVersion
}

type versionData struct {
	SpecName           []byte
	ImplName           []byte
	AuthoringVersion   uint32
	SpecVersion        uint32
	ImplVersion        uint32
	APIItems           []APIItem
	TransactionVersion uint32
}

// Encode returns the SCALE encoding of the Version
func (vd *VersionData) Encode() ([]byte, error) {
	info := versionData{
		SpecName:           vd.specName,
		ImplName:           vd.implName,
		AuthoringVersion:   vd.authoringVersion,
		SpecVersion:        vd.specVersion,
		ImplVersion:        vd.implVersion,
		APIItems:           vd.apiItems,
		TransactionVersion: vd.transactionVersion,
	}

	enc, err := scale.Marshal(info)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

// Decode to scale decode []byte to VersionAPI struct
func (vd *VersionData) Decode(in []byte) error {
	var info versionData
	err := scale.Unmarshal(in, &info)
	if err != nil {
		return err
	}

	vd.specName = info.SpecName
	vd.implName = info.ImplName
	vd.authoringVersion = info.AuthoringVersion
	vd.specVersion = info.SpecVersion
	vd.implVersion = info.ImplVersion
	vd.apiItems = info.APIItems
	vd.transactionVersion = info.TransactionVersion

	return nil
}
