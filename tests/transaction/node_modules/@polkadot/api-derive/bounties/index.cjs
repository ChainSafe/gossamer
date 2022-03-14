"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _bounties = require("./bounties.cjs");

Object.keys(_bounties).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _bounties[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _bounties[key];
    }
  });
});