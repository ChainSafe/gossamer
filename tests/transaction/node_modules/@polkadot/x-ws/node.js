// Copyright 2017-2022 @polkadot/x-ws authors & contributors
// SPDX-License-Identifier: Apache-2.0
import ws from 'websocket';
import { extractGlobal } from '@polkadot/x-global';
export { packageInfo } from "./packageInfo.js";
export const WebSocket = extractGlobal('WebSocket', ws.w3cwebsocket);