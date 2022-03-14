// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isOnObject } from "./helpers.js";
export const isPromise = isOnObject('catch', 'then');