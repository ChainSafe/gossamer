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

type Version struct {
	Spec_name         []byte
	Impl_name         []byte
	Authoring_version int32
	Spec_version      int32
	Impl_version      int32
}

var (
	CoreVersion                               = "Core_version"
	CoreInitializeBlock                       = "Core_initialize_block"
	CoreExecuteBlock                          = "Core_execute_block"
	TaggedTransactionQueueValidateTransaction = "TaggedTransactionQueue_validate_transaction"
	BabeApiConfiguration                      = "BabeApi_configuration"
	AuraApiAuthorities							= "AuraApi_authorities"
	BlockBuilderInherentExtrinsics            = "BlockBuilder_inherent_extrinsics"
	BlockBuilderApplyExtrinsic                = "BlockBuilder_apply_extrinsic"
	BlockBuilderFinalizeBlock                 = "BlockBuilder_finalize_block"
)
