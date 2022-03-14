"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.GenericExtrinsicPayloadUnknown = void 0;

var _typesCodec = require("@polkadot/types-codec");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name GenericExtrinsicPayloadUnknown
 * @description
 * A default handler for payloads where the version is not known (default throw)
 */
class GenericExtrinsicPayloadUnknown extends _typesCodec.Struct {
  constructor(registry, value) {
    let {
      version = 0
    } = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : {};
    super(registry, {});
    throw new Error(`Unsupported extrinsic payload version ${version}`);
  }

}

exports.GenericExtrinsicPayloadUnknown = GenericExtrinsicPayloadUnknown;