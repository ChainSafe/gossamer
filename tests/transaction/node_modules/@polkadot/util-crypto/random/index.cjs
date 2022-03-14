"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
Object.defineProperty(exports, "randomAsHex", {
  enumerable: true,
  get: function () {
    return _asU8a.randomAsHex;
  }
});
Object.defineProperty(exports, "randomAsNumber", {
  enumerable: true,
  get: function () {
    return _asNumber.randomAsNumber;
  }
});
Object.defineProperty(exports, "randomAsU8a", {
  enumerable: true,
  get: function () {
    return _asU8a.randomAsU8a;
  }
});

var _asNumber = require("./asNumber.cjs");

var _asU8a = require("./asU8a.cjs");