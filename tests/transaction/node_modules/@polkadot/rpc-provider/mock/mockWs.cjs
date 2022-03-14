"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.TEST_WS_URL = void 0;
exports.mockWs = mockWs;

var _mockSocket = require("mock-socket");

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/rpc-provider authors & contributors
// SPDX-License-Identifier: Apache-2.0
global.WebSocket = _mockSocket.WebSocket;
const TEST_WS_URL = 'ws://localhost:9955'; // should be JSONRPC def return

exports.TEST_WS_URL = TEST_WS_URL;

function createError(_ref) {
  let {
    error: {
      code,
      message
    },
    id
  } = _ref;
  return {
    error: {
      code,
      message
    },
    id,
    jsonrpc: '2.0'
  };
} // should be JSONRPC def return


function createReply(_ref2) {
  let {
    id,
    reply: {
      result
    }
  } = _ref2;
  return {
    id,
    jsonrpc: '2.0',
    result
  };
} // scope definition returned


function mockWs(requests) {
  let wsUrl = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : TEST_WS_URL;
  const server = new _mockSocket.Server(wsUrl);
  let requestCount = 0;
  const scope = {
    body: {},
    done: () => {
      server.stop(() => {// ignore
      });
    },
    requests: 0,
    server
  };
  server.on('connection', socket => {
    socket.on('message', body => {
      const request = requests[requestCount];
      const response = request.error ? createError(request) : createReply(request);
      scope.body[request.method] = body;
      requestCount++;
      socket.send((0, _util.stringify)(response));
    });
  });
  return scope;
}