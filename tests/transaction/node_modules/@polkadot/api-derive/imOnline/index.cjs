"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _receivedHeartbeats = require("./receivedHeartbeats.cjs");

Object.keys(_receivedHeartbeats).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _receivedHeartbeats[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _receivedHeartbeats[key];
    }
  });
});