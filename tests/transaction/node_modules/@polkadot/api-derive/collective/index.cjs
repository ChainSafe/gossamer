"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _members = require("./members.cjs");

Object.keys(_members).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _members[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _members[key];
    }
  });
});

var _prime = require("./prime.cjs");

Object.keys(_prime).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _prime[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _prime[key];
    }
  });
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