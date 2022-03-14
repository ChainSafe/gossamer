"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.insecureRandomValues = insecureRandomValues;
// Copyright 2017-2022 @polkadot/x-randomvalues authors & contributors
// SPDX-License-Identifier: Apache-2.0
// Adapted from https://github.com/LinusU/react-native-get-random-values/blob/85f48393821c23b83b89a8177f56d3a81dc8b733/index.js
// Copyright (c) 2018, 2020 Linus Unnebäck
// SPDX-License-Identifier: MIT
let warned = false;

function insecureRandomValues(arr) {
  if (!warned) {
    console.warn('Using an insecure random number generator, this should only happen when running in a debugger without support for crypto');
    warned = true;
  }

  let r = 0;

  for (let i = 0; i < arr.length; i++) {
    if ((i & 0b11) === 0) {
      r = Math.random() * 0x100000000;
    }

    arr[i] = r >>> ((i & 0b11) << 3) & 0xff;
  }

  return arr;
}