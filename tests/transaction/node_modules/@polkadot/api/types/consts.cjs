"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _consts = require("@polkadot/api-base/types/consts");

Object.keys(_consts).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _consts[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _consts[key];
    }
  });
});