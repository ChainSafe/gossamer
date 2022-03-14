"use strict";

var _util = require("@polkadot/util");

var _packageInfo = require("@polkadot/wasm-crypto-asmjs/packageInfo");

var _packageInfo2 = require("@polkadot/wasm-crypto-wasm/packageInfo");

var _packageInfo3 = require("./packageInfo.cjs");

// Copyright 2017-2021 @polkadot/wasm-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
(0, _util.detectPackage)(_packageInfo3.packageInfo, typeof __dirname !== 'undefined' && __dirname, [_packageInfo.packageInfo, _packageInfo2.packageInfo]);