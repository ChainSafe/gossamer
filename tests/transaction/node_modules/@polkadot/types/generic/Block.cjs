"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.GenericBlock = void 0;

var _typesCodec = require("@polkadot/types-codec");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name GenericBlock
 * @description
 * A block encoded with header and extrinsics
 */
class GenericBlock extends _typesCodec.Struct {
  constructor(registry, value) {
    super(registry, {
      header: 'Header',
      // eslint-disable-next-line sort-keys
      extrinsics: 'Vec<Extrinsic>'
    }, value);
  }
  /**
   * @description Encodes a content [[Hash]] for the block
   */


  get contentHash() {
    return this.registry.hash(this.toU8a());
  }
  /**
   * @description The [[Extrinsic]] contained in the block
   */


  get extrinsics() {
    return this.getT('extrinsics');
  }
  /**
   * @description Block/header [[Hash]]
   */


  get hash() {
    return this.header.hash;
  }
  /**
   * @description The [[Header]] of the block
   */


  get header() {
    return this.getT('header');
  }

}

exports.GenericBlock = GenericBlock;