// Copyright 2017-2022 @polkadot/types-known authors & contributors
// SPDX-License-Identifier: Apache-2.0
import centrifugeChain from "./centrifuge-chain.js";
import kusama from "./kusama.js";
import node from "./node.js";
import nodeTemplate from "./node-template.js";
import polkadot from "./polkadot.js";
import rococo from "./rococo.js";
import shell from "./shell.js";
import statemint from "./statemint.js";
import westend from "./westend.js"; // Type overrides for specific spec types & versions as given in runtimeVersion

const typesSpec = {
  'centrifuge-chain': centrifugeChain,
  kusama,
  node,
  'node-template': nodeTemplate,
  polkadot,
  rococo,
  shell,
  statemine: statemint,
  statemint,
  westend,
  westmint: statemint
};
export default typesSpec;