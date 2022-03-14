"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});

var _candidates = require("./candidates.cjs");

Object.keys(_candidates).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _candidates[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _candidates[key];
    }
  });
});

var _info = require("./info.cjs");

Object.keys(_info).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _info[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _info[key];
    }
  });
});

var _member = require("./member.cjs");

Object.keys(_member).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _member[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _member[key];
    }
  });
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