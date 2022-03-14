"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _proposals = require("./proposals.cjs");

Object.keys(_proposals).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _proposals[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _proposals[key];
    }
  });
});