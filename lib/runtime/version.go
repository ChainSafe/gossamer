// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import "github.com/ChainSafe/gossamer/pkg/scale"

// Version represents the data returned by runtime call core_version
type Version interface {
	GetSpecName() []byte
	GetImplName() []byte
	GetAuthoringVersion() uint32
	GetSpecVersion() uint32
	GetImplVersion() uint32
	GetAPIItems() []APIItem
	GetTransactionVersion() uint32
	// Encode returns the scale encoding of the version.
	// Note this one cannot be replaced by scale.Marshal
	// or the Version interface pointer would be marshalled,
	// instead of the version implementation.
	Encode() (encoded []byte, err error)
}

// APIItem struct to hold runtime API Name and Version
type APIItem struct {
	Name [8]byte
	Ver  uint32
}

// LegacyVersionData is the runtime version info returned by legacy runtimes
type LegacyVersionData struct {
	SpecName         []byte
	ImplName         []byte
	AuthoringVersion uint32
	SpecVersion      uint32
	ImplVersion      uint32
	APIItems         []APIItem
}

// GetSpecName returns the spec name
func (lvd *LegacyVersionData) GetSpecName() []byte {
	return lvd.SpecName
}

// GetImplName returns the implementation name
func (lvd *LegacyVersionData) GetImplName() []byte {
	return lvd.ImplName
}

// GetAuthoringVersion returns the authoring version
func (lvd *LegacyVersionData) GetAuthoringVersion() uint32 {
	return lvd.AuthoringVersion
}

// GetSpecVersion returns the spec version
func (lvd *LegacyVersionData) GetSpecVersion() uint32 {
	return lvd.SpecVersion
}

// GetImplVersion returns the implementation version
func (lvd *LegacyVersionData) GetImplVersion() uint32 {
	return lvd.ImplVersion
}

// GetAPIItems returns the API items
func (lvd *LegacyVersionData) GetAPIItems() []APIItem {
	return lvd.APIItems
}

// GetTransactionVersion returns the transaction version
func (*LegacyVersionData) GetTransactionVersion() uint32 {
	return 0
}

// Encode returns the scale encoding of the version.
func (lvd *LegacyVersionData) Encode() (encoded []byte, err error) {
	return scale.Marshal(*lvd)
}

// VersionData is the runtime version info returned by v0.8 runtimes
type VersionData struct {
	SpecName           []byte
	ImplName           []byte
	AuthoringVersion   uint32
	SpecVersion        uint32
	ImplVersion        uint32
	APIItems           []APIItem
	TransactionVersion uint32
}

// GetSpecName returns the spec name
func (vd *VersionData) GetSpecName() []byte {
	return vd.SpecName
}

// GetImplName returns the implementation name
func (vd *VersionData) GetImplName() []byte {
	return vd.ImplName
}

// GetAuthoringVersion returns the authoring version
func (vd *VersionData) GetAuthoringVersion() uint32 {
	return vd.AuthoringVersion
}

// GetSpecVersion returns the spec version
func (vd *VersionData) GetSpecVersion() uint32 {
	return vd.SpecVersion
}

// GetImplVersion returns the implementation version
func (vd *VersionData) GetImplVersion() uint32 {
	return vd.ImplVersion
}

// GetAPIItems returns the API items
func (vd *VersionData) GetAPIItems() []APIItem {
	return vd.APIItems
}

// GetTransactionVersion returns the transaction version
func (vd *VersionData) GetTransactionVersion() uint32 {
	return vd.TransactionVersion
}

// Encode returns the scale encoding of the version.
func (vd *VersionData) Encode() (encoded []byte, err error) {
	return scale.Marshal(*vd)
}
