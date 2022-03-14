"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _storage = require("@polkadot/api-base/types/storage");

Object.keys(_storage).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _storage[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _storage[key];
    }
  });
});