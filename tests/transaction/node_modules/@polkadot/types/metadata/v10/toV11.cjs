"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.toV11 = toV11;

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0

/** @internal */
function toV11(registry, _ref) {
  let {
    modules
  } = _ref;
  return registry.createTypeUnsafe('MetadataV11', [{
    // This is new in V11, pass V0 here - something non-existing, telling the API to use
    // the fallback for this information (on-chain detection)
    extrinsic: {
      signedExtensions: [],
      version: 0
    },
    modules
  }]);
}