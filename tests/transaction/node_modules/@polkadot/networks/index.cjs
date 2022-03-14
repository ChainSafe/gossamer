"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
var _exportNames = {
  packageInfo: true
};
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});

var _interfaces = require("./interfaces.cjs");

Object.keys(_interfaces).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (Object.prototype.hasOwnProperty.call(_exportNames, key)) return;
  if (key in exports && exports[key] === _interfaces[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _interfaces[key];
    }
  });
});

var _packageInfo = require("./packageInfo.cjs");