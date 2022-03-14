"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.XCM_MAPPINGS = void 0;
exports.mapXcmTypes = mapXcmTypes;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types-create authors & contributors
// SPDX-License-Identifier: Apache-2.0
const XCM_MAPPINGS = ['AssetInstance', 'Fungibility', 'Junction', 'Junctions', 'MultiAsset', 'MultiAssetFilter', 'MultiLocation', 'Response', 'WildFungibility', 'WildMultiAsset', 'Xcm', 'XcmError', 'XcmOrder'];
exports.XCM_MAPPINGS = XCM_MAPPINGS;

function mapXcmTypes(version) {
  return XCM_MAPPINGS.reduce((all, key) => (0, _util.objectSpread)(all, {
    [key]: `${key}${version}`
  }), {});
}