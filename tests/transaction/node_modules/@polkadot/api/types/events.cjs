"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _events = require("@polkadot/api-base/types/events");

Object.keys(_events).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _events[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _events[key];
    }
  });
});