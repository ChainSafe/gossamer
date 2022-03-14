// Copyright 2017-2022 @polkadot/types-create authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { objectSpread } from '@polkadot/util';
export const XCM_MAPPINGS = ['AssetInstance', 'Fungibility', 'Junction', 'Junctions', 'MultiAsset', 'MultiAssetFilter', 'MultiLocation', 'Response', 'WildFungibility', 'WildMultiAsset', 'Xcm', 'XcmError', 'XcmOrder'];
export function mapXcmTypes(version) {
  return XCM_MAPPINGS.reduce((all, key) => objectSpread(all, {
    [key]: `${key}${version}`
  }), {});
}