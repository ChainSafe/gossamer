// Copyright 2017-2022 @polkadot/x-fetch authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { fetch } from '@polkadot/x-fetch';
import { exposeGlobal } from '@polkadot/x-global';
exposeGlobal('fetch', fetch);