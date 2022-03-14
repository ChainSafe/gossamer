// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { objectSpread } from '@polkadot/util';
import * as defs from "./definitions.js";
const jsonrpc = {};
Object.keys(defs).forEach(s => Object.entries(defs[s].rpc || {}).forEach(([method, def]) => {
  // allow for section overrides
  const section = def.aliasSection || s;

  if (!jsonrpc[section]) {
    jsonrpc[section] = {};
  }

  jsonrpc[section][method] = objectSpread({}, def, {
    isSubscription: !!def.pubsub,
    jsonrpc: `${section}_${method}`,
    method,
    section
  });
}));
export default jsonrpc;