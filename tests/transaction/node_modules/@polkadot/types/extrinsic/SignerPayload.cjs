"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.GenericSignerPayload = void 0;

var _typesCodec = require("@polkadot/types-codec");

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
const knownTypes = {
  address: 'Address',
  blockHash: 'Hash',
  blockNumber: 'BlockNumber',
  era: 'ExtrinsicEra',
  genesisHash: 'Hash',
  method: 'Call',
  nonce: 'Compact<Index>',
  runtimeVersion: 'RuntimeVersion',
  signedExtensions: 'Vec<Text>',
  tip: 'Compact<Balance>',
  version: 'u8'
};
/**
 * @name GenericSignerPayload
 * @description
 * A generic signer payload that can be used for serialization between API and signer
 */

class GenericSignerPayload extends _typesCodec.Struct {
  #extraTypes;

  constructor(registry, value) {
    const extensionTypes = (0, _util.objectSpread)({}, registry.getSignedExtensionTypes(), registry.getSignedExtensionExtra());
    super(registry, (0, _util.objectSpread)({}, extensionTypes, knownTypes), value);
    this.#extraTypes = {};

    const getter = key => this.get(key); // add all extras that are not in the base types


    for (const [key, type] of Object.entries(extensionTypes)) {
      if (!knownTypes[key]) {
        this.#extraTypes[key] = type;
      }

      (0, _util.objectProperty)(this, key, getter);
    }
  }

  get address() {
    return this.getT('address');
  }

  get blockHash() {
    return this.getT('blockHash');
  }

  get blockNumber() {
    return this.getT('blockNumber');
  }

  get era() {
    return this.getT('era');
  }

  get genesisHash() {
    return this.getT('genesisHash');
  }

  get method() {
    return this.getT('method');
  }

  get nonce() {
    return this.getT('nonce');
  }

  get runtimeVersion() {
    return this.getT('runtimeVersion');
  }

  get signedExtensions() {
    return this.getT('signedExtensions');
  }

  get tip() {
    return this.getT('tip');
  }

  get version() {
    return this.getT('version');
  }
  /**
   * @description Creates an representation of the structure as an ISignerPayload JSON
   */


  toPayload() {
    const result = {};
    const keys = Object.keys(this.#extraTypes); // add any explicit overrides we may have

    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      const value = this.get(key);
      const isOption = value instanceof _typesCodec.Option; // Don't include Option.isNone

      if (!isOption || value.isSome) {
        result[key] = value.toHex();
      }
    }

    return (0, _util.objectSpread)(result, {
      // the known defaults as managed explicitly and has different
      // formatting in cases, e.g. we mostly expose a hex format here
      address: this.address.toString(),
      blockHash: this.blockHash.toHex(),
      blockNumber: this.blockNumber.toHex(),
      era: this.era.toHex(),
      genesisHash: this.genesisHash.toHex(),
      method: this.method.toHex(),
      nonce: this.nonce.toHex(),
      signedExtensions: this.signedExtensions.map(e => e.toString()),
      specVersion: this.runtimeVersion.specVersion.toHex(),
      tip: this.tip.toHex(),
      transactionVersion: this.runtimeVersion.transactionVersion.toHex(),
      version: this.version.toNumber()
    });
  }
  /**
   * @description Creates a representation of the payload in raw Exrinsic form
   */


  toRaw() {
    const payload = this.toPayload();
    const data = (0, _util.u8aToHex)(this.registry.createTypeUnsafe('ExtrinsicPayload', [payload, {
      version: payload.version
    }]) // NOTE Explicitly pass the bare flag so the method is encoded un-prefixed (non-decodable, for signing only)
    .toU8a({
      method: true
    }));
    return {
      address: payload.address,
      data,
      type: 'payload'
    };
  }

}

exports.GenericSignerPayload = GenericSignerPayload;