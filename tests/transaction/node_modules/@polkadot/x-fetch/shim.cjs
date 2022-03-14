"use strict";

var _xFetch = require("@polkadot/x-fetch");

var _xGlobal = require("@polkadot/x-global");

// Copyright 2017-2022 @polkadot/x-fetch authors & contributors
// SPDX-License-Identifier: Apache-2.0
(0, _xGlobal.exposeGlobal)('fetch', _xFetch.fetch);