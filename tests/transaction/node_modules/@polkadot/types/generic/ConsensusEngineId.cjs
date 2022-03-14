"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.GenericConsensusEngineId = exports.CID_POW = exports.CID_GRPA = exports.CID_BABE = exports.CID_AURA = void 0;

var _typesCodec = require("@polkadot/types-codec");

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
const CID_AURA = (0, _util.stringToU8a)('aura');
exports.CID_AURA = CID_AURA;
const CID_BABE = (0, _util.stringToU8a)('BABE');
exports.CID_BABE = CID_BABE;
const CID_GRPA = (0, _util.stringToU8a)('FRNK');
exports.CID_GRPA = CID_GRPA;
const CID_POW = (0, _util.stringToU8a)('pow_');
exports.CID_POW = CID_POW;

function getAuraAuthor(registry, bytes, sessionValidators) {
  return sessionValidators[registry.createTypeUnsafe('RawAuraPreDigest', [bytes.toU8a(true)]).slotNumber.mod(new _util.BN(sessionValidators.length)).toNumber()];
}

function getBabeAuthor(registry, bytes, sessionValidators) {
  const digest = registry.createTypeUnsafe('RawBabePreDigestCompat', [bytes.toU8a(true)]);
  return sessionValidators[digest.value.toNumber()];
}

function getBytesAsAuthor(registry, bytes) {
  return registry.createTypeUnsafe('AccountId', [bytes]);
}
/**
 * @name GenericConsensusEngineId
 * @description
 * A 4-byte identifier identifying the engine
 */


class GenericConsensusEngineId extends _typesCodec.U8aFixed {
  constructor(registry, value) {
    super(registry, (0, _util.isNumber)(value) ? (0, _util.bnToU8a)(value, {
      isLe: false
    }) : value, 32);
  }
  /**
   * @description `true` if the engine matches aura
   */


  get isAura() {
    return this.eq(CID_AURA);
  }
  /**
   * @description `true` is the engine matches babe
   */


  get isBabe() {
    return this.eq(CID_BABE);
  }
  /**
   * @description `true` is the engine matches grandpa
   */


  get isGrandpa() {
    return this.eq(CID_GRPA);
  }
  /**
   * @description `true` is the engine matches pow
   */


  get isPow() {
    return this.eq(CID_POW);
  }
  /**
   * @description From the input bytes, decode into an author
   */


  extractAuthor(bytes, sessionValidators) {
    if (sessionValidators !== null && sessionValidators !== void 0 && sessionValidators.length) {
      if (this.isAura) {
        return getAuraAuthor(this.registry, bytes, sessionValidators);
      } else if (this.isBabe) {
        return getBabeAuthor(this.registry, bytes, sessionValidators);
      }
    } // For pow & Moonbeam, the bytes are the actual author


    if (this.isPow || bytes.length === 20) {
      return getBytesAsAuthor(this.registry, bytes);
    }

    return undefined;
  }
  /**
   * @description Converts the Object to to a human-friendly JSON, with additional fields, expansion and formatting of information
   */


  toHuman() {
    return this.toString();
  }
  /**
   * @description Returns the base runtime type name for this instance
   */


  toRawType() {
    return 'ConsensusEngineId';
  }
  /**
   * @description Override the default toString to return a 4-byte string
   */


  toString() {
    return this.isAscii ? (0, _util.u8aToString)(this) : (0, _util.u8aToHex)(this);
  }

}

exports.GenericConsensusEngineId = GenericConsensusEngineId;