// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { U8aFixed } from '@polkadot/types-codec';
import { BN, bnToU8a, isNumber, stringToU8a, u8aToHex, u8aToString } from '@polkadot/util';
export const CID_AURA = stringToU8a('aura');
export const CID_BABE = stringToU8a('BABE');
export const CID_GRPA = stringToU8a('FRNK');
export const CID_POW = stringToU8a('pow_');

function getAuraAuthor(registry, bytes, sessionValidators) {
  return sessionValidators[registry.createTypeUnsafe('RawAuraPreDigest', [bytes.toU8a(true)]).slotNumber.mod(new BN(sessionValidators.length)).toNumber()];
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


export class GenericConsensusEngineId extends U8aFixed {
  constructor(registry, value) {
    super(registry, isNumber(value) ? bnToU8a(value, {
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
    return this.isAscii ? u8aToString(this) : u8aToHex(this);
  }

}