"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _types = require("./address/types.cjs");

Object.keys(_types).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _types[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _types[key];
    }
  });
});

var _types2 = require("./json/types.cjs");

Object.keys(_types2).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _types2[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _types2[key];
    }
  });
});