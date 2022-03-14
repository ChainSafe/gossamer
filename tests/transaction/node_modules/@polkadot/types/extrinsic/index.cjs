"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
var _exportNames = {
  GenericExtrinsic: true,
  GenericExtrinsicEra: true,
  GenericMortalEra: true,
  GenericImmortalEra: true,
  GenericExtrinsicPayload: true,
  GenericExtrinsicPayloadUnknown: true,
  GenericExtrinsicUnknown: true,
  GenericSignerPayload: true
};
Object.defineProperty(exports, "GenericExtrinsic", {
  enumerable: true,
  get: function () {
    return _Extrinsic.GenericExtrinsic;
  }
});
Object.defineProperty(exports, "GenericExtrinsicEra", {
  enumerable: true,
  get: function () {
    return _ExtrinsicEra.GenericExtrinsicEra;
  }
});
Object.defineProperty(exports, "GenericExtrinsicPayload", {
  enumerable: true,
  get: function () {
    return _ExtrinsicPayload.GenericExtrinsicPayload;
  }
});
Object.defineProperty(exports, "GenericExtrinsicPayloadUnknown", {
  enumerable: true,
  get: function () {
    return _ExtrinsicPayloadUnknown.GenericExtrinsicPayloadUnknown;
  }
});
Object.defineProperty(exports, "GenericExtrinsicUnknown", {
  enumerable: true,
  get: function () {
    return _ExtrinsicUnknown.GenericExtrinsicUnknown;
  }
});
Object.defineProperty(exports, "GenericImmortalEra", {
  enumerable: true,
  get: function () {
    return _ExtrinsicEra.ImmortalEra;
  }
});
Object.defineProperty(exports, "GenericMortalEra", {
  enumerable: true,
  get: function () {
    return _ExtrinsicEra.MortalEra;
  }
});
Object.defineProperty(exports, "GenericSignerPayload", {
  enumerable: true,
  get: function () {
    return _SignerPayload.GenericSignerPayload;
  }
});

var _Extrinsic = require("./Extrinsic.cjs");

var _ExtrinsicEra = require("./ExtrinsicEra.cjs");

var _ExtrinsicPayload = require("./ExtrinsicPayload.cjs");

var _ExtrinsicPayloadUnknown = require("./ExtrinsicPayloadUnknown.cjs");

var _ExtrinsicUnknown = require("./ExtrinsicUnknown.cjs");

var _SignerPayload = require("./SignerPayload.cjs");

var _index = require("./v4/index.cjs");

Object.keys(_index).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (Object.prototype.hasOwnProperty.call(_exportNames, key)) return;
  if (key in exports && exports[key] === _index[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _index[key];
    }
  });
});