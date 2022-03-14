"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.TextEncoder = void 0;

// Copyright 2017-2022 @polkadot/x-textencoder authors & contributors
// SPDX-License-Identifier: Apache-2.0
// This is very limited, only handling Ascii values
class TextEncoder {
  encode(value) {
    const u8a = new Uint8Array(value.length);

    for (let i = 0; i < value.length; i++) {
      u8a[i] = value.charCodeAt(i);
    }

    return u8a;
  }

}

exports.TextEncoder = TextEncoder;