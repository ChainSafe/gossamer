// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
// Originally from https://github.com/polkadot-js/extension/pull/743
import { u8aConcat } from "./concat.js";
import { u8aEq } from "./eq.js";
import { u8aToU8a } from "./toU8a.js";
export const U8A_WRAP_ETHEREUM = u8aToU8a('\x19Ethereum Signed Message:\n');
export const U8A_WRAP_PREFIX = u8aToU8a('<Bytes>');
export const U8A_WRAP_POSTFIX = u8aToU8a('</Bytes>');
const WRAP_LEN = U8A_WRAP_PREFIX.length + U8A_WRAP_POSTFIX.length;
export function u8aIsWrapped(u8a, withEthereum) {
  return u8a.length >= WRAP_LEN && u8aEq(u8a.subarray(0, U8A_WRAP_PREFIX.length), U8A_WRAP_PREFIX) && u8aEq(u8a.slice(-U8A_WRAP_POSTFIX.length), U8A_WRAP_POSTFIX) || withEthereum && u8a.length >= U8A_WRAP_ETHEREUM.length && u8aEq(u8a.subarray(0, U8A_WRAP_ETHEREUM.length), U8A_WRAP_ETHEREUM);
}
export function u8aUnwrapBytes(bytes) {
  const u8a = u8aToU8a(bytes); // we don't want to unwrap Ethereum-style wraps

  return u8aIsWrapped(u8a, false) ? u8a.subarray(U8A_WRAP_PREFIX.length, u8a.length - U8A_WRAP_POSTFIX.length) : u8a;
}
export function u8aWrapBytes(bytes) {
  const u8a = u8aToU8a(bytes); // if Ethereum-wrapping, we don't add our wrapping bytes

  return u8aIsWrapped(u8a, true) ? u8a : u8aConcat(U8A_WRAP_PREFIX, u8a, U8A_WRAP_POSTFIX);
}