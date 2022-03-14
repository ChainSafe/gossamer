// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { callMethod } from "./helpers.js"; // We are re-exporting these from here to ensure that *.d.ts generation is correct

export const members = callMethod('members', []);