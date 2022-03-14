"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _submittable = require("@polkadot/api-base/types/submittable");

Object.keys(_submittable).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _submittable[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _submittable[key];
    }
  });
});