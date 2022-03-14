(function (global, factory) {
  typeof exports === 'object' && typeof module !== 'undefined' ? factory(exports) :
  typeof define === 'function' && define.amd ? define(['exports'], factory) :
  (global = typeof globalThis !== 'undefined' ? globalThis : global || self, factory(global.polkadotUtil = {}));
})(this, (function (exports) { 'use strict';

  const global = window;

  const packageInfo = {
    name: '@polkadot/util',
    path: new URL('.', (typeof document === 'undefined' && typeof location === 'undefined' ? new (require('u' + 'rl').URL)('file:' + __filename).href : typeof document === 'undefined' ? location.href : (document.currentScript && document.currentScript.src || new URL('bundle-polkadot-util.js', document.baseURI).href))).pathname,
    type: 'esm',
    version: '8.4.1'
  };

  function arrayChunk(array, chunkSize) {
    const outputSize = Math.ceil(array.length / chunkSize);
    const output = Array(outputSize);
    for (let i = 0; i < outputSize; i++) {
      const offset = i * chunkSize;
      output[i] = array.slice(offset, offset + chunkSize);
    }
    return output;
  }

  function isNull(value) {
    return value === null;
  }

  function isUndefined(value) {
    return typeof value === 'undefined';
  }

  function arrayFilter(array, allowNulls = true) {
    return array.filter(v => !isUndefined(v) && (allowNulls || !isNull(v)));
  }

  function arrayFlatten(arrays) {
    let size = 0;
    for (let i = 0; i < arrays.length; i++) {
      size += arrays[i].length;
    }
    const output = new Array(size);
    let i = -1;
    for (let j = 0; j < arrays.length; j++) {
      const a = arrays[j];
      for (let e = 0; e < a.length; e++) {
        output[++i] = a[e];
      }
    }
    return output;
  }

  function isFunction(value) {
    return typeof value === 'function';
  }

  function assert(condition, message) {
    if (!condition) {
      throw new Error(isFunction(message) ? message() : message);
    }
  }
  function assertReturn(value, message) {
    assert(!isUndefined(value) && !isNull(value), message);
    return value;
  }
  function assertUnreachable(x) {
    throw new Error(`This codepath should be unreachable. Unhandled input: ${x}`);
  }

  function arrayRange(size, startAt = 0) {
    assert(size > 0, 'Expected non-zero, positive number as a range size');
    const result = new Array(size);
    for (let i = 0; i < size; i++) {
      result[i] = i + startAt;
    }
    return result;
  }

  function arrayShuffle(input) {
    const result = input.slice();
    let curr = result.length;
    if (curr === 1) {
      return result;
    }
    while (curr !== 0) {
      const rand = Math.floor(Math.random() * curr);
      curr--;
      [result[curr], result[rand]] = [result[rand], result[curr]];
    }
    return result;
  }

  function arrayZip(keys, values) {
    const result = new Array(keys.length);
    for (let i = 0; i < keys.length; i++) {
      result[i] = [keys[i], values[i]];
    }
    return result;
  }

  function evaluateThis(fn) {
    return fn('return this');
  }
  const xglobal = typeof globalThis !== 'undefined' ? globalThis : typeof global !== 'undefined' ? global : typeof self !== 'undefined' ? self : typeof window !== 'undefined' ? window : evaluateThis(Function);
  function extractGlobal(name, fallback) {
    return typeof xglobal[name] === 'undefined' ? fallback : xglobal[name];
  }

  const BigInt = typeof xglobal.BigInt === 'function' && typeof xglobal.BigInt.asIntN === 'function' ? xglobal.BigInt : () => Number.NaN;

  const _0n = BigInt(0);
  const _1n = BigInt(1);
  const _1Mn = BigInt(1000000);
  const _1Bn = BigInt(1000000000);
  const _1Qn = _1Bn * _1Bn;
  const _2pow53n = BigInt(Number.MAX_SAFE_INTEGER);

  function objectKeys(value) {
    return Object.keys(value);
  }

  function objectSpread(dest, ...sources) {
    for (let i = 0; i < sources.length; i++) {
      const src = sources[i];
      if (src) {
        const keys = objectKeys(src);
        for (let j = 0; j < keys.length; j++) {
          const key = keys[j];
          dest[key] = src[key];
        }
      }
    }
    return dest;
  }

  const U8_MAX = BigInt(256);
  const U16_MAX = BigInt(256 * 256);
  function xor(input) {
    const result = new Uint8Array(input.length);
    const dvI = new DataView(input.buffer, input.byteOffset);
    const dvO = new DataView(result.buffer);
    const mod = input.length % 2;
    const length = input.length - mod;
    for (let i = 0; i < length; i += 2) {
      dvO.setUint16(i, dvI.getUint16(i) ^ 0xffff);
    }
    if (mod) {
      dvO.setUint8(length, dvI.getUint8(length) ^ 0xff);
    }
    return result;
  }
  function toBigInt(input) {
    const dvI = new DataView(input.buffer, input.byteOffset);
    const mod = input.length % 2;
    const length = input.length - mod;
    let result = BigInt(0);
    for (let i = 0; i < length; i += 2) {
      result = result * U16_MAX + BigInt(dvI.getUint16(i));
    }
    if (mod) {
      result = result * U8_MAX + BigInt(dvI.getUint8(length));
    }
    return result;
  }
  function u8aToBigInt(value, options = {}) {
    if (!value || !value.length) {
      return BigInt(0);
    }
    const {
      isLe,
      isNegative
    } = objectSpread({
      isLe: true,
      isNegative: false
    }, options);
    const u8a = isLe ? value.reverse() : value;
    return isNegative ? toBigInt(xor(u8a)) * -_1n - _1n : toBigInt(u8a);
  }

  const U8_TO_HEX = new Array(256);
  const U16_TO_HEX = new Array(256 * 256);
  const HEX_TO_U8 = {};
  const HEX_TO_U16 = {};
  for (let n = 0; n < 256; n++) {
    const hex = n.toString(16).padStart(2, '0');
    U8_TO_HEX[n] = hex;
    HEX_TO_U8[hex] = n;
  }
  for (let i = 0; i < 256; i++) {
    for (let j = 0; j < 256; j++) {
      const hex = U8_TO_HEX[i] + U8_TO_HEX[j];
      const n = i << 8 | j;
      U16_TO_HEX[n] = hex;
      HEX_TO_U16[hex] = n;
    }
  }

  const REGEX_HEX_PREFIXED = /^0x[\da-fA-F]+$/;
  const REGEX_HEX_NOPREFIX = /^[\da-fA-F]+$/;
  function isHex(value, bitLength = -1, ignoreLength) {
    return typeof value === 'string' && (value === '0x' || REGEX_HEX_PREFIXED.test(value)) && (bitLength === -1 ? ignoreLength || value.length % 2 === 0 : value.length === 2 + Math.ceil(bitLength / 4));
  }

  function hexStripPrefix(value) {
    if (!value || value === '0x') {
      return '';
    } else if (REGEX_HEX_PREFIXED.test(value)) {
      return value.substr(2);
    } else if (REGEX_HEX_NOPREFIX.test(value)) {
      return value;
    }
    throw new Error(`Expected hex value to convert, found '${value}'`);
  }

  function hexToU8a(_value, bitLength = -1) {
    if (!_value) {
      return new Uint8Array();
    }
    const value = hexStripPrefix(_value).toLowerCase();
    const valLength = value.length / 2;
    const endLength = Math.ceil(bitLength === -1 ? valLength : bitLength / 8);
    const result = new Uint8Array(endLength);
    const offset = endLength > valLength ? endLength - valLength : 0;
    const dv = new DataView(result.buffer, offset);
    const mod = (endLength - offset) % 2;
    const length = endLength - offset - mod;
    for (let i = 0; i < length; i += 2) {
      dv.setUint16(i, HEX_TO_U16[value.substr(i * 2, 4)]);
    }
    if (mod) {
      dv.setUint8(length, HEX_TO_U8[value.substr(value.length - 2, 2)]);
    }
    return result;
  }

  function hexToBigInt(value, options = {}) {
    return !value || value === '0x' ? BigInt(0) : u8aToBigInt(hexToU8a(value), objectSpread({
      isLe: false,
      isNegative: false
    }, options));
  }

  var commonjsGlobal = typeof globalThis !== 'undefined' ? globalThis : typeof window !== 'undefined' ? window : typeof global !== 'undefined' ? global : typeof self !== 'undefined' ? self : {};

  var bn = {exports: {}};

  (function (module) {
  (function (module, exports) {
    function assert (val, msg) {
      if (!val) throw new Error(msg || 'Assertion failed');
    }
    function inherits (ctor, superCtor) {
      ctor.super_ = superCtor;
      var TempCtor = function () {};
      TempCtor.prototype = superCtor.prototype;
      ctor.prototype = new TempCtor();
      ctor.prototype.constructor = ctor;
    }
    function BN (number, base, endian) {
      if (BN.isBN(number)) {
        return number;
      }
      this.negative = 0;
      this.words = null;
      this.length = 0;
      this.red = null;
      if (number !== null) {
        if (base === 'le' || base === 'be') {
          endian = base;
          base = 10;
        }
        this._init(number || 0, base || 10, endian || 'be');
      }
    }
    if (typeof module === 'object') {
      module.exports = BN;
    } else {
      exports.BN = BN;
    }
    BN.BN = BN;
    BN.wordSize = 26;
    var Buffer;
    try {
      if (typeof window !== 'undefined' && typeof window.Buffer !== 'undefined') {
        Buffer = window.Buffer;
      } else {
        Buffer = require('buffer').Buffer;
      }
    } catch (e) {
    }
    BN.isBN = function isBN (num) {
      if (num instanceof BN) {
        return true;
      }
      return num !== null && typeof num === 'object' &&
        num.constructor.wordSize === BN.wordSize && Array.isArray(num.words);
    };
    BN.max = function max (left, right) {
      if (left.cmp(right) > 0) return left;
      return right;
    };
    BN.min = function min (left, right) {
      if (left.cmp(right) < 0) return left;
      return right;
    };
    BN.prototype._init = function init (number, base, endian) {
      if (typeof number === 'number') {
        return this._initNumber(number, base, endian);
      }
      if (typeof number === 'object') {
        return this._initArray(number, base, endian);
      }
      if (base === 'hex') {
        base = 16;
      }
      assert(base === (base | 0) && base >= 2 && base <= 36);
      number = number.toString().replace(/\s+/g, '');
      var start = 0;
      if (number[0] === '-') {
        start++;
        this.negative = 1;
      }
      if (start < number.length) {
        if (base === 16) {
          this._parseHex(number, start, endian);
        } else {
          this._parseBase(number, base, start);
          if (endian === 'le') {
            this._initArray(this.toArray(), base, endian);
          }
        }
      }
    };
    BN.prototype._initNumber = function _initNumber (number, base, endian) {
      if (number < 0) {
        this.negative = 1;
        number = -number;
      }
      if (number < 0x4000000) {
        this.words = [number & 0x3ffffff];
        this.length = 1;
      } else if (number < 0x10000000000000) {
        this.words = [
          number & 0x3ffffff,
          (number / 0x4000000) & 0x3ffffff
        ];
        this.length = 2;
      } else {
        assert(number < 0x20000000000000);
        this.words = [
          number & 0x3ffffff,
          (number / 0x4000000) & 0x3ffffff,
          1
        ];
        this.length = 3;
      }
      if (endian !== 'le') return;
      this._initArray(this.toArray(), base, endian);
    };
    BN.prototype._initArray = function _initArray (number, base, endian) {
      assert(typeof number.length === 'number');
      if (number.length <= 0) {
        this.words = [0];
        this.length = 1;
        return this;
      }
      this.length = Math.ceil(number.length / 3);
      this.words = new Array(this.length);
      for (var i = 0; i < this.length; i++) {
        this.words[i] = 0;
      }
      var j, w;
      var off = 0;
      if (endian === 'be') {
        for (i = number.length - 1, j = 0; i >= 0; i -= 3) {
          w = number[i] | (number[i - 1] << 8) | (number[i - 2] << 16);
          this.words[j] |= (w << off) & 0x3ffffff;
          this.words[j + 1] = (w >>> (26 - off)) & 0x3ffffff;
          off += 24;
          if (off >= 26) {
            off -= 26;
            j++;
          }
        }
      } else if (endian === 'le') {
        for (i = 0, j = 0; i < number.length; i += 3) {
          w = number[i] | (number[i + 1] << 8) | (number[i + 2] << 16);
          this.words[j] |= (w << off) & 0x3ffffff;
          this.words[j + 1] = (w >>> (26 - off)) & 0x3ffffff;
          off += 24;
          if (off >= 26) {
            off -= 26;
            j++;
          }
        }
      }
      return this._strip();
    };
    function parseHex4Bits (string, index) {
      var c = string.charCodeAt(index);
      if (c >= 48 && c <= 57) {
        return c - 48;
      } else if (c >= 65 && c <= 70) {
        return c - 55;
      } else if (c >= 97 && c <= 102) {
        return c - 87;
      } else {
        assert(false, 'Invalid character in ' + string);
      }
    }
    function parseHexByte (string, lowerBound, index) {
      var r = parseHex4Bits(string, index);
      if (index - 1 >= lowerBound) {
        r |= parseHex4Bits(string, index - 1) << 4;
      }
      return r;
    }
    BN.prototype._parseHex = function _parseHex (number, start, endian) {
      this.length = Math.ceil((number.length - start) / 6);
      this.words = new Array(this.length);
      for (var i = 0; i < this.length; i++) {
        this.words[i] = 0;
      }
      var off = 0;
      var j = 0;
      var w;
      if (endian === 'be') {
        for (i = number.length - 1; i >= start; i -= 2) {
          w = parseHexByte(number, start, i) << off;
          this.words[j] |= w & 0x3ffffff;
          if (off >= 18) {
            off -= 18;
            j += 1;
            this.words[j] |= w >>> 26;
          } else {
            off += 8;
          }
        }
      } else {
        var parseLength = number.length - start;
        for (i = parseLength % 2 === 0 ? start + 1 : start; i < number.length; i += 2) {
          w = parseHexByte(number, start, i) << off;
          this.words[j] |= w & 0x3ffffff;
          if (off >= 18) {
            off -= 18;
            j += 1;
            this.words[j] |= w >>> 26;
          } else {
            off += 8;
          }
        }
      }
      this._strip();
    };
    function parseBase (str, start, end, mul) {
      var r = 0;
      var b = 0;
      var len = Math.min(str.length, end);
      for (var i = start; i < len; i++) {
        var c = str.charCodeAt(i) - 48;
        r *= mul;
        if (c >= 49) {
          b = c - 49 + 0xa;
        } else if (c >= 17) {
          b = c - 17 + 0xa;
        } else {
          b = c;
        }
        assert(c >= 0 && b < mul, 'Invalid character');
        r += b;
      }
      return r;
    }
    BN.prototype._parseBase = function _parseBase (number, base, start) {
      this.words = [0];
      this.length = 1;
      for (var limbLen = 0, limbPow = 1; limbPow <= 0x3ffffff; limbPow *= base) {
        limbLen++;
      }
      limbLen--;
      limbPow = (limbPow / base) | 0;
      var total = number.length - start;
      var mod = total % limbLen;
      var end = Math.min(total, total - mod) + start;
      var word = 0;
      for (var i = start; i < end; i += limbLen) {
        word = parseBase(number, i, i + limbLen, base);
        this.imuln(limbPow);
        if (this.words[0] + word < 0x4000000) {
          this.words[0] += word;
        } else {
          this._iaddn(word);
        }
      }
      if (mod !== 0) {
        var pow = 1;
        word = parseBase(number, i, number.length, base);
        for (i = 0; i < mod; i++) {
          pow *= base;
        }
        this.imuln(pow);
        if (this.words[0] + word < 0x4000000) {
          this.words[0] += word;
        } else {
          this._iaddn(word);
        }
      }
      this._strip();
    };
    BN.prototype.copy = function copy (dest) {
      dest.words = new Array(this.length);
      for (var i = 0; i < this.length; i++) {
        dest.words[i] = this.words[i];
      }
      dest.length = this.length;
      dest.negative = this.negative;
      dest.red = this.red;
    };
    function move (dest, src) {
      dest.words = src.words;
      dest.length = src.length;
      dest.negative = src.negative;
      dest.red = src.red;
    }
    BN.prototype._move = function _move (dest) {
      move(dest, this);
    };
    BN.prototype.clone = function clone () {
      var r = new BN(null);
      this.copy(r);
      return r;
    };
    BN.prototype._expand = function _expand (size) {
      while (this.length < size) {
        this.words[this.length++] = 0;
      }
      return this;
    };
    BN.prototype._strip = function strip () {
      while (this.length > 1 && this.words[this.length - 1] === 0) {
        this.length--;
      }
      return this._normSign();
    };
    BN.prototype._normSign = function _normSign () {
      if (this.length === 1 && this.words[0] === 0) {
        this.negative = 0;
      }
      return this;
    };
    if (typeof Symbol !== 'undefined' && typeof Symbol.for === 'function') {
      try {
        BN.prototype[Symbol.for('nodejs.util.inspect.custom')] = inspect;
      } catch (e) {
        BN.prototype.inspect = inspect;
      }
    } else {
      BN.prototype.inspect = inspect;
    }
    function inspect () {
      return (this.red ? '<BN-R: ' : '<BN: ') + this.toString(16) + '>';
    }
    var zeros = [
      '',
      '0',
      '00',
      '000',
      '0000',
      '00000',
      '000000',
      '0000000',
      '00000000',
      '000000000',
      '0000000000',
      '00000000000',
      '000000000000',
      '0000000000000',
      '00000000000000',
      '000000000000000',
      '0000000000000000',
      '00000000000000000',
      '000000000000000000',
      '0000000000000000000',
      '00000000000000000000',
      '000000000000000000000',
      '0000000000000000000000',
      '00000000000000000000000',
      '000000000000000000000000',
      '0000000000000000000000000'
    ];
    var groupSizes = [
      0, 0,
      25, 16, 12, 11, 10, 9, 8,
      8, 7, 7, 7, 7, 6, 6,
      6, 6, 6, 6, 6, 5, 5,
      5, 5, 5, 5, 5, 5, 5,
      5, 5, 5, 5, 5, 5, 5
    ];
    var groupBases = [
      0, 0,
      33554432, 43046721, 16777216, 48828125, 60466176, 40353607, 16777216,
      43046721, 10000000, 19487171, 35831808, 62748517, 7529536, 11390625,
      16777216, 24137569, 34012224, 47045881, 64000000, 4084101, 5153632,
      6436343, 7962624, 9765625, 11881376, 14348907, 17210368, 20511149,
      24300000, 28629151, 33554432, 39135393, 45435424, 52521875, 60466176
    ];
    BN.prototype.toString = function toString (base, padding) {
      base = base || 10;
      padding = padding | 0 || 1;
      var out;
      if (base === 16 || base === 'hex') {
        out = '';
        var off = 0;
        var carry = 0;
        for (var i = 0; i < this.length; i++) {
          var w = this.words[i];
          var word = (((w << off) | carry) & 0xffffff).toString(16);
          carry = (w >>> (24 - off)) & 0xffffff;
          if (carry !== 0 || i !== this.length - 1) {
            out = zeros[6 - word.length] + word + out;
          } else {
            out = word + out;
          }
          off += 2;
          if (off >= 26) {
            off -= 26;
            i--;
          }
        }
        if (carry !== 0) {
          out = carry.toString(16) + out;
        }
        while (out.length % padding !== 0) {
          out = '0' + out;
        }
        if (this.negative !== 0) {
          out = '-' + out;
        }
        return out;
      }
      if (base === (base | 0) && base >= 2 && base <= 36) {
        var groupSize = groupSizes[base];
        var groupBase = groupBases[base];
        out = '';
        var c = this.clone();
        c.negative = 0;
        while (!c.isZero()) {
          var r = c.modrn(groupBase).toString(base);
          c = c.idivn(groupBase);
          if (!c.isZero()) {
            out = zeros[groupSize - r.length] + r + out;
          } else {
            out = r + out;
          }
        }
        if (this.isZero()) {
          out = '0' + out;
        }
        while (out.length % padding !== 0) {
          out = '0' + out;
        }
        if (this.negative !== 0) {
          out = '-' + out;
        }
        return out;
      }
      assert(false, 'Base should be between 2 and 36');
    };
    BN.prototype.toNumber = function toNumber () {
      var ret = this.words[0];
      if (this.length === 2) {
        ret += this.words[1] * 0x4000000;
      } else if (this.length === 3 && this.words[2] === 0x01) {
        ret += 0x10000000000000 + (this.words[1] * 0x4000000);
      } else if (this.length > 2) {
        assert(false, 'Number can only safely store up to 53 bits');
      }
      return (this.negative !== 0) ? -ret : ret;
    };
    BN.prototype.toJSON = function toJSON () {
      return this.toString(16, 2);
    };
    if (Buffer) {
      BN.prototype.toBuffer = function toBuffer (endian, length) {
        return this.toArrayLike(Buffer, endian, length);
      };
    }
    BN.prototype.toArray = function toArray (endian, length) {
      return this.toArrayLike(Array, endian, length);
    };
    var allocate = function allocate (ArrayType, size) {
      if (ArrayType.allocUnsafe) {
        return ArrayType.allocUnsafe(size);
      }
      return new ArrayType(size);
    };
    BN.prototype.toArrayLike = function toArrayLike (ArrayType, endian, length) {
      this._strip();
      var byteLength = this.byteLength();
      var reqLength = length || Math.max(1, byteLength);
      assert(byteLength <= reqLength, 'byte array longer than desired length');
      assert(reqLength > 0, 'Requested array length <= 0');
      var res = allocate(ArrayType, reqLength);
      var postfix = endian === 'le' ? 'LE' : 'BE';
      this['_toArrayLike' + postfix](res, byteLength);
      return res;
    };
    BN.prototype._toArrayLikeLE = function _toArrayLikeLE (res, byteLength) {
      var position = 0;
      var carry = 0;
      for (var i = 0, shift = 0; i < this.length; i++) {
        var word = (this.words[i] << shift) | carry;
        res[position++] = word & 0xff;
        if (position < res.length) {
          res[position++] = (word >> 8) & 0xff;
        }
        if (position < res.length) {
          res[position++] = (word >> 16) & 0xff;
        }
        if (shift === 6) {
          if (position < res.length) {
            res[position++] = (word >> 24) & 0xff;
          }
          carry = 0;
          shift = 0;
        } else {
          carry = word >>> 24;
          shift += 2;
        }
      }
      if (position < res.length) {
        res[position++] = carry;
        while (position < res.length) {
          res[position++] = 0;
        }
      }
    };
    BN.prototype._toArrayLikeBE = function _toArrayLikeBE (res, byteLength) {
      var position = res.length - 1;
      var carry = 0;
      for (var i = 0, shift = 0; i < this.length; i++) {
        var word = (this.words[i] << shift) | carry;
        res[position--] = word & 0xff;
        if (position >= 0) {
          res[position--] = (word >> 8) & 0xff;
        }
        if (position >= 0) {
          res[position--] = (word >> 16) & 0xff;
        }
        if (shift === 6) {
          if (position >= 0) {
            res[position--] = (word >> 24) & 0xff;
          }
          carry = 0;
          shift = 0;
        } else {
          carry = word >>> 24;
          shift += 2;
        }
      }
      if (position >= 0) {
        res[position--] = carry;
        while (position >= 0) {
          res[position--] = 0;
        }
      }
    };
    if (Math.clz32) {
      BN.prototype._countBits = function _countBits (w) {
        return 32 - Math.clz32(w);
      };
    } else {
      BN.prototype._countBits = function _countBits (w) {
        var t = w;
        var r = 0;
        if (t >= 0x1000) {
          r += 13;
          t >>>= 13;
        }
        if (t >= 0x40) {
          r += 7;
          t >>>= 7;
        }
        if (t >= 0x8) {
          r += 4;
          t >>>= 4;
        }
        if (t >= 0x02) {
          r += 2;
          t >>>= 2;
        }
        return r + t;
      };
    }
    BN.prototype._zeroBits = function _zeroBits (w) {
      if (w === 0) return 26;
      var t = w;
      var r = 0;
      if ((t & 0x1fff) === 0) {
        r += 13;
        t >>>= 13;
      }
      if ((t & 0x7f) === 0) {
        r += 7;
        t >>>= 7;
      }
      if ((t & 0xf) === 0) {
        r += 4;
        t >>>= 4;
      }
      if ((t & 0x3) === 0) {
        r += 2;
        t >>>= 2;
      }
      if ((t & 0x1) === 0) {
        r++;
      }
      return r;
    };
    BN.prototype.bitLength = function bitLength () {
      var w = this.words[this.length - 1];
      var hi = this._countBits(w);
      return (this.length - 1) * 26 + hi;
    };
    function toBitArray (num) {
      var w = new Array(num.bitLength());
      for (var bit = 0; bit < w.length; bit++) {
        var off = (bit / 26) | 0;
        var wbit = bit % 26;
        w[bit] = (num.words[off] >>> wbit) & 0x01;
      }
      return w;
    }
    BN.prototype.zeroBits = function zeroBits () {
      if (this.isZero()) return 0;
      var r = 0;
      for (var i = 0; i < this.length; i++) {
        var b = this._zeroBits(this.words[i]);
        r += b;
        if (b !== 26) break;
      }
      return r;
    };
    BN.prototype.byteLength = function byteLength () {
      return Math.ceil(this.bitLength() / 8);
    };
    BN.prototype.toTwos = function toTwos (width) {
      if (this.negative !== 0) {
        return this.abs().inotn(width).iaddn(1);
      }
      return this.clone();
    };
    BN.prototype.fromTwos = function fromTwos (width) {
      if (this.testn(width - 1)) {
        return this.notn(width).iaddn(1).ineg();
      }
      return this.clone();
    };
    BN.prototype.isNeg = function isNeg () {
      return this.negative !== 0;
    };
    BN.prototype.neg = function neg () {
      return this.clone().ineg();
    };
    BN.prototype.ineg = function ineg () {
      if (!this.isZero()) {
        this.negative ^= 1;
      }
      return this;
    };
    BN.prototype.iuor = function iuor (num) {
      while (this.length < num.length) {
        this.words[this.length++] = 0;
      }
      for (var i = 0; i < num.length; i++) {
        this.words[i] = this.words[i] | num.words[i];
      }
      return this._strip();
    };
    BN.prototype.ior = function ior (num) {
      assert((this.negative | num.negative) === 0);
      return this.iuor(num);
    };
    BN.prototype.or = function or (num) {
      if (this.length > num.length) return this.clone().ior(num);
      return num.clone().ior(this);
    };
    BN.prototype.uor = function uor (num) {
      if (this.length > num.length) return this.clone().iuor(num);
      return num.clone().iuor(this);
    };
    BN.prototype.iuand = function iuand (num) {
      var b;
      if (this.length > num.length) {
        b = num;
      } else {
        b = this;
      }
      for (var i = 0; i < b.length; i++) {
        this.words[i] = this.words[i] & num.words[i];
      }
      this.length = b.length;
      return this._strip();
    };
    BN.prototype.iand = function iand (num) {
      assert((this.negative | num.negative) === 0);
      return this.iuand(num);
    };
    BN.prototype.and = function and (num) {
      if (this.length > num.length) return this.clone().iand(num);
      return num.clone().iand(this);
    };
    BN.prototype.uand = function uand (num) {
      if (this.length > num.length) return this.clone().iuand(num);
      return num.clone().iuand(this);
    };
    BN.prototype.iuxor = function iuxor (num) {
      var a;
      var b;
      if (this.length > num.length) {
        a = this;
        b = num;
      } else {
        a = num;
        b = this;
      }
      for (var i = 0; i < b.length; i++) {
        this.words[i] = a.words[i] ^ b.words[i];
      }
      if (this !== a) {
        for (; i < a.length; i++) {
          this.words[i] = a.words[i];
        }
      }
      this.length = a.length;
      return this._strip();
    };
    BN.prototype.ixor = function ixor (num) {
      assert((this.negative | num.negative) === 0);
      return this.iuxor(num);
    };
    BN.prototype.xor = function xor (num) {
      if (this.length > num.length) return this.clone().ixor(num);
      return num.clone().ixor(this);
    };
    BN.prototype.uxor = function uxor (num) {
      if (this.length > num.length) return this.clone().iuxor(num);
      return num.clone().iuxor(this);
    };
    BN.prototype.inotn = function inotn (width) {
      assert(typeof width === 'number' && width >= 0);
      var bytesNeeded = Math.ceil(width / 26) | 0;
      var bitsLeft = width % 26;
      this._expand(bytesNeeded);
      if (bitsLeft > 0) {
        bytesNeeded--;
      }
      for (var i = 0; i < bytesNeeded; i++) {
        this.words[i] = ~this.words[i] & 0x3ffffff;
      }
      if (bitsLeft > 0) {
        this.words[i] = ~this.words[i] & (0x3ffffff >> (26 - bitsLeft));
      }
      return this._strip();
    };
    BN.prototype.notn = function notn (width) {
      return this.clone().inotn(width);
    };
    BN.prototype.setn = function setn (bit, val) {
      assert(typeof bit === 'number' && bit >= 0);
      var off = (bit / 26) | 0;
      var wbit = bit % 26;
      this._expand(off + 1);
      if (val) {
        this.words[off] = this.words[off] | (1 << wbit);
      } else {
        this.words[off] = this.words[off] & ~(1 << wbit);
      }
      return this._strip();
    };
    BN.prototype.iadd = function iadd (num) {
      var r;
      if (this.negative !== 0 && num.negative === 0) {
        this.negative = 0;
        r = this.isub(num);
        this.negative ^= 1;
        return this._normSign();
      } else if (this.negative === 0 && num.negative !== 0) {
        num.negative = 0;
        r = this.isub(num);
        num.negative = 1;
        return r._normSign();
      }
      var a, b;
      if (this.length > num.length) {
        a = this;
        b = num;
      } else {
        a = num;
        b = this;
      }
      var carry = 0;
      for (var i = 0; i < b.length; i++) {
        r = (a.words[i] | 0) + (b.words[i] | 0) + carry;
        this.words[i] = r & 0x3ffffff;
        carry = r >>> 26;
      }
      for (; carry !== 0 && i < a.length; i++) {
        r = (a.words[i] | 0) + carry;
        this.words[i] = r & 0x3ffffff;
        carry = r >>> 26;
      }
      this.length = a.length;
      if (carry !== 0) {
        this.words[this.length] = carry;
        this.length++;
      } else if (a !== this) {
        for (; i < a.length; i++) {
          this.words[i] = a.words[i];
        }
      }
      return this;
    };
    BN.prototype.add = function add (num) {
      var res;
      if (num.negative !== 0 && this.negative === 0) {
        num.negative = 0;
        res = this.sub(num);
        num.negative ^= 1;
        return res;
      } else if (num.negative === 0 && this.negative !== 0) {
        this.negative = 0;
        res = num.sub(this);
        this.negative = 1;
        return res;
      }
      if (this.length > num.length) return this.clone().iadd(num);
      return num.clone().iadd(this);
    };
    BN.prototype.isub = function isub (num) {
      if (num.negative !== 0) {
        num.negative = 0;
        var r = this.iadd(num);
        num.negative = 1;
        return r._normSign();
      } else if (this.negative !== 0) {
        this.negative = 0;
        this.iadd(num);
        this.negative = 1;
        return this._normSign();
      }
      var cmp = this.cmp(num);
      if (cmp === 0) {
        this.negative = 0;
        this.length = 1;
        this.words[0] = 0;
        return this;
      }
      var a, b;
      if (cmp > 0) {
        a = this;
        b = num;
      } else {
        a = num;
        b = this;
      }
      var carry = 0;
      for (var i = 0; i < b.length; i++) {
        r = (a.words[i] | 0) - (b.words[i] | 0) + carry;
        carry = r >> 26;
        this.words[i] = r & 0x3ffffff;
      }
      for (; carry !== 0 && i < a.length; i++) {
        r = (a.words[i] | 0) + carry;
        carry = r >> 26;
        this.words[i] = r & 0x3ffffff;
      }
      if (carry === 0 && i < a.length && a !== this) {
        for (; i < a.length; i++) {
          this.words[i] = a.words[i];
        }
      }
      this.length = Math.max(this.length, i);
      if (a !== this) {
        this.negative = 1;
      }
      return this._strip();
    };
    BN.prototype.sub = function sub (num) {
      return this.clone().isub(num);
    };
    function smallMulTo (self, num, out) {
      out.negative = num.negative ^ self.negative;
      var len = (self.length + num.length) | 0;
      out.length = len;
      len = (len - 1) | 0;
      var a = self.words[0] | 0;
      var b = num.words[0] | 0;
      var r = a * b;
      var lo = r & 0x3ffffff;
      var carry = (r / 0x4000000) | 0;
      out.words[0] = lo;
      for (var k = 1; k < len; k++) {
        var ncarry = carry >>> 26;
        var rword = carry & 0x3ffffff;
        var maxJ = Math.min(k, num.length - 1);
        for (var j = Math.max(0, k - self.length + 1); j <= maxJ; j++) {
          var i = (k - j) | 0;
          a = self.words[i] | 0;
          b = num.words[j] | 0;
          r = a * b + rword;
          ncarry += (r / 0x4000000) | 0;
          rword = r & 0x3ffffff;
        }
        out.words[k] = rword | 0;
        carry = ncarry | 0;
      }
      if (carry !== 0) {
        out.words[k] = carry | 0;
      } else {
        out.length--;
      }
      return out._strip();
    }
    var comb10MulTo = function comb10MulTo (self, num, out) {
      var a = self.words;
      var b = num.words;
      var o = out.words;
      var c = 0;
      var lo;
      var mid;
      var hi;
      var a0 = a[0] | 0;
      var al0 = a0 & 0x1fff;
      var ah0 = a0 >>> 13;
      var a1 = a[1] | 0;
      var al1 = a1 & 0x1fff;
      var ah1 = a1 >>> 13;
      var a2 = a[2] | 0;
      var al2 = a2 & 0x1fff;
      var ah2 = a2 >>> 13;
      var a3 = a[3] | 0;
      var al3 = a3 & 0x1fff;
      var ah3 = a3 >>> 13;
      var a4 = a[4] | 0;
      var al4 = a4 & 0x1fff;
      var ah4 = a4 >>> 13;
      var a5 = a[5] | 0;
      var al5 = a5 & 0x1fff;
      var ah5 = a5 >>> 13;
      var a6 = a[6] | 0;
      var al6 = a6 & 0x1fff;
      var ah6 = a6 >>> 13;
      var a7 = a[7] | 0;
      var al7 = a7 & 0x1fff;
      var ah7 = a7 >>> 13;
      var a8 = a[8] | 0;
      var al8 = a8 & 0x1fff;
      var ah8 = a8 >>> 13;
      var a9 = a[9] | 0;
      var al9 = a9 & 0x1fff;
      var ah9 = a9 >>> 13;
      var b0 = b[0] | 0;
      var bl0 = b0 & 0x1fff;
      var bh0 = b0 >>> 13;
      var b1 = b[1] | 0;
      var bl1 = b1 & 0x1fff;
      var bh1 = b1 >>> 13;
      var b2 = b[2] | 0;
      var bl2 = b2 & 0x1fff;
      var bh2 = b2 >>> 13;
      var b3 = b[3] | 0;
      var bl3 = b3 & 0x1fff;
      var bh3 = b3 >>> 13;
      var b4 = b[4] | 0;
      var bl4 = b4 & 0x1fff;
      var bh4 = b4 >>> 13;
      var b5 = b[5] | 0;
      var bl5 = b5 & 0x1fff;
      var bh5 = b5 >>> 13;
      var b6 = b[6] | 0;
      var bl6 = b6 & 0x1fff;
      var bh6 = b6 >>> 13;
      var b7 = b[7] | 0;
      var bl7 = b7 & 0x1fff;
      var bh7 = b7 >>> 13;
      var b8 = b[8] | 0;
      var bl8 = b8 & 0x1fff;
      var bh8 = b8 >>> 13;
      var b9 = b[9] | 0;
      var bl9 = b9 & 0x1fff;
      var bh9 = b9 >>> 13;
      out.negative = self.negative ^ num.negative;
      out.length = 19;
      lo = Math.imul(al0, bl0);
      mid = Math.imul(al0, bh0);
      mid = (mid + Math.imul(ah0, bl0)) | 0;
      hi = Math.imul(ah0, bh0);
      var w0 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w0 >>> 26)) | 0;
      w0 &= 0x3ffffff;
      lo = Math.imul(al1, bl0);
      mid = Math.imul(al1, bh0);
      mid = (mid + Math.imul(ah1, bl0)) | 0;
      hi = Math.imul(ah1, bh0);
      lo = (lo + Math.imul(al0, bl1)) | 0;
      mid = (mid + Math.imul(al0, bh1)) | 0;
      mid = (mid + Math.imul(ah0, bl1)) | 0;
      hi = (hi + Math.imul(ah0, bh1)) | 0;
      var w1 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w1 >>> 26)) | 0;
      w1 &= 0x3ffffff;
      lo = Math.imul(al2, bl0);
      mid = Math.imul(al2, bh0);
      mid = (mid + Math.imul(ah2, bl0)) | 0;
      hi = Math.imul(ah2, bh0);
      lo = (lo + Math.imul(al1, bl1)) | 0;
      mid = (mid + Math.imul(al1, bh1)) | 0;
      mid = (mid + Math.imul(ah1, bl1)) | 0;
      hi = (hi + Math.imul(ah1, bh1)) | 0;
      lo = (lo + Math.imul(al0, bl2)) | 0;
      mid = (mid + Math.imul(al0, bh2)) | 0;
      mid = (mid + Math.imul(ah0, bl2)) | 0;
      hi = (hi + Math.imul(ah0, bh2)) | 0;
      var w2 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w2 >>> 26)) | 0;
      w2 &= 0x3ffffff;
      lo = Math.imul(al3, bl0);
      mid = Math.imul(al3, bh0);
      mid = (mid + Math.imul(ah3, bl0)) | 0;
      hi = Math.imul(ah3, bh0);
      lo = (lo + Math.imul(al2, bl1)) | 0;
      mid = (mid + Math.imul(al2, bh1)) | 0;
      mid = (mid + Math.imul(ah2, bl1)) | 0;
      hi = (hi + Math.imul(ah2, bh1)) | 0;
      lo = (lo + Math.imul(al1, bl2)) | 0;
      mid = (mid + Math.imul(al1, bh2)) | 0;
      mid = (mid + Math.imul(ah1, bl2)) | 0;
      hi = (hi + Math.imul(ah1, bh2)) | 0;
      lo = (lo + Math.imul(al0, bl3)) | 0;
      mid = (mid + Math.imul(al0, bh3)) | 0;
      mid = (mid + Math.imul(ah0, bl3)) | 0;
      hi = (hi + Math.imul(ah0, bh3)) | 0;
      var w3 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w3 >>> 26)) | 0;
      w3 &= 0x3ffffff;
      lo = Math.imul(al4, bl0);
      mid = Math.imul(al4, bh0);
      mid = (mid + Math.imul(ah4, bl0)) | 0;
      hi = Math.imul(ah4, bh0);
      lo = (lo + Math.imul(al3, bl1)) | 0;
      mid = (mid + Math.imul(al3, bh1)) | 0;
      mid = (mid + Math.imul(ah3, bl1)) | 0;
      hi = (hi + Math.imul(ah3, bh1)) | 0;
      lo = (lo + Math.imul(al2, bl2)) | 0;
      mid = (mid + Math.imul(al2, bh2)) | 0;
      mid = (mid + Math.imul(ah2, bl2)) | 0;
      hi = (hi + Math.imul(ah2, bh2)) | 0;
      lo = (lo + Math.imul(al1, bl3)) | 0;
      mid = (mid + Math.imul(al1, bh3)) | 0;
      mid = (mid + Math.imul(ah1, bl3)) | 0;
      hi = (hi + Math.imul(ah1, bh3)) | 0;
      lo = (lo + Math.imul(al0, bl4)) | 0;
      mid = (mid + Math.imul(al0, bh4)) | 0;
      mid = (mid + Math.imul(ah0, bl4)) | 0;
      hi = (hi + Math.imul(ah0, bh4)) | 0;
      var w4 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w4 >>> 26)) | 0;
      w4 &= 0x3ffffff;
      lo = Math.imul(al5, bl0);
      mid = Math.imul(al5, bh0);
      mid = (mid + Math.imul(ah5, bl0)) | 0;
      hi = Math.imul(ah5, bh0);
      lo = (lo + Math.imul(al4, bl1)) | 0;
      mid = (mid + Math.imul(al4, bh1)) | 0;
      mid = (mid + Math.imul(ah4, bl1)) | 0;
      hi = (hi + Math.imul(ah4, bh1)) | 0;
      lo = (lo + Math.imul(al3, bl2)) | 0;
      mid = (mid + Math.imul(al3, bh2)) | 0;
      mid = (mid + Math.imul(ah3, bl2)) | 0;
      hi = (hi + Math.imul(ah3, bh2)) | 0;
      lo = (lo + Math.imul(al2, bl3)) | 0;
      mid = (mid + Math.imul(al2, bh3)) | 0;
      mid = (mid + Math.imul(ah2, bl3)) | 0;
      hi = (hi + Math.imul(ah2, bh3)) | 0;
      lo = (lo + Math.imul(al1, bl4)) | 0;
      mid = (mid + Math.imul(al1, bh4)) | 0;
      mid = (mid + Math.imul(ah1, bl4)) | 0;
      hi = (hi + Math.imul(ah1, bh4)) | 0;
      lo = (lo + Math.imul(al0, bl5)) | 0;
      mid = (mid + Math.imul(al0, bh5)) | 0;
      mid = (mid + Math.imul(ah0, bl5)) | 0;
      hi = (hi + Math.imul(ah0, bh5)) | 0;
      var w5 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w5 >>> 26)) | 0;
      w5 &= 0x3ffffff;
      lo = Math.imul(al6, bl0);
      mid = Math.imul(al6, bh0);
      mid = (mid + Math.imul(ah6, bl0)) | 0;
      hi = Math.imul(ah6, bh0);
      lo = (lo + Math.imul(al5, bl1)) | 0;
      mid = (mid + Math.imul(al5, bh1)) | 0;
      mid = (mid + Math.imul(ah5, bl1)) | 0;
      hi = (hi + Math.imul(ah5, bh1)) | 0;
      lo = (lo + Math.imul(al4, bl2)) | 0;
      mid = (mid + Math.imul(al4, bh2)) | 0;
      mid = (mid + Math.imul(ah4, bl2)) | 0;
      hi = (hi + Math.imul(ah4, bh2)) | 0;
      lo = (lo + Math.imul(al3, bl3)) | 0;
      mid = (mid + Math.imul(al3, bh3)) | 0;
      mid = (mid + Math.imul(ah3, bl3)) | 0;
      hi = (hi + Math.imul(ah3, bh3)) | 0;
      lo = (lo + Math.imul(al2, bl4)) | 0;
      mid = (mid + Math.imul(al2, bh4)) | 0;
      mid = (mid + Math.imul(ah2, bl4)) | 0;
      hi = (hi + Math.imul(ah2, bh4)) | 0;
      lo = (lo + Math.imul(al1, bl5)) | 0;
      mid = (mid + Math.imul(al1, bh5)) | 0;
      mid = (mid + Math.imul(ah1, bl5)) | 0;
      hi = (hi + Math.imul(ah1, bh5)) | 0;
      lo = (lo + Math.imul(al0, bl6)) | 0;
      mid = (mid + Math.imul(al0, bh6)) | 0;
      mid = (mid + Math.imul(ah0, bl6)) | 0;
      hi = (hi + Math.imul(ah0, bh6)) | 0;
      var w6 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w6 >>> 26)) | 0;
      w6 &= 0x3ffffff;
      lo = Math.imul(al7, bl0);
      mid = Math.imul(al7, bh0);
      mid = (mid + Math.imul(ah7, bl0)) | 0;
      hi = Math.imul(ah7, bh0);
      lo = (lo + Math.imul(al6, bl1)) | 0;
      mid = (mid + Math.imul(al6, bh1)) | 0;
      mid = (mid + Math.imul(ah6, bl1)) | 0;
      hi = (hi + Math.imul(ah6, bh1)) | 0;
      lo = (lo + Math.imul(al5, bl2)) | 0;
      mid = (mid + Math.imul(al5, bh2)) | 0;
      mid = (mid + Math.imul(ah5, bl2)) | 0;
      hi = (hi + Math.imul(ah5, bh2)) | 0;
      lo = (lo + Math.imul(al4, bl3)) | 0;
      mid = (mid + Math.imul(al4, bh3)) | 0;
      mid = (mid + Math.imul(ah4, bl3)) | 0;
      hi = (hi + Math.imul(ah4, bh3)) | 0;
      lo = (lo + Math.imul(al3, bl4)) | 0;
      mid = (mid + Math.imul(al3, bh4)) | 0;
      mid = (mid + Math.imul(ah3, bl4)) | 0;
      hi = (hi + Math.imul(ah3, bh4)) | 0;
      lo = (lo + Math.imul(al2, bl5)) | 0;
      mid = (mid + Math.imul(al2, bh5)) | 0;
      mid = (mid + Math.imul(ah2, bl5)) | 0;
      hi = (hi + Math.imul(ah2, bh5)) | 0;
      lo = (lo + Math.imul(al1, bl6)) | 0;
      mid = (mid + Math.imul(al1, bh6)) | 0;
      mid = (mid + Math.imul(ah1, bl6)) | 0;
      hi = (hi + Math.imul(ah1, bh6)) | 0;
      lo = (lo + Math.imul(al0, bl7)) | 0;
      mid = (mid + Math.imul(al0, bh7)) | 0;
      mid = (mid + Math.imul(ah0, bl7)) | 0;
      hi = (hi + Math.imul(ah0, bh7)) | 0;
      var w7 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w7 >>> 26)) | 0;
      w7 &= 0x3ffffff;
      lo = Math.imul(al8, bl0);
      mid = Math.imul(al8, bh0);
      mid = (mid + Math.imul(ah8, bl0)) | 0;
      hi = Math.imul(ah8, bh0);
      lo = (lo + Math.imul(al7, bl1)) | 0;
      mid = (mid + Math.imul(al7, bh1)) | 0;
      mid = (mid + Math.imul(ah7, bl1)) | 0;
      hi = (hi + Math.imul(ah7, bh1)) | 0;
      lo = (lo + Math.imul(al6, bl2)) | 0;
      mid = (mid + Math.imul(al6, bh2)) | 0;
      mid = (mid + Math.imul(ah6, bl2)) | 0;
      hi = (hi + Math.imul(ah6, bh2)) | 0;
      lo = (lo + Math.imul(al5, bl3)) | 0;
      mid = (mid + Math.imul(al5, bh3)) | 0;
      mid = (mid + Math.imul(ah5, bl3)) | 0;
      hi = (hi + Math.imul(ah5, bh3)) | 0;
      lo = (lo + Math.imul(al4, bl4)) | 0;
      mid = (mid + Math.imul(al4, bh4)) | 0;
      mid = (mid + Math.imul(ah4, bl4)) | 0;
      hi = (hi + Math.imul(ah4, bh4)) | 0;
      lo = (lo + Math.imul(al3, bl5)) | 0;
      mid = (mid + Math.imul(al3, bh5)) | 0;
      mid = (mid + Math.imul(ah3, bl5)) | 0;
      hi = (hi + Math.imul(ah3, bh5)) | 0;
      lo = (lo + Math.imul(al2, bl6)) | 0;
      mid = (mid + Math.imul(al2, bh6)) | 0;
      mid = (mid + Math.imul(ah2, bl6)) | 0;
      hi = (hi + Math.imul(ah2, bh6)) | 0;
      lo = (lo + Math.imul(al1, bl7)) | 0;
      mid = (mid + Math.imul(al1, bh7)) | 0;
      mid = (mid + Math.imul(ah1, bl7)) | 0;
      hi = (hi + Math.imul(ah1, bh7)) | 0;
      lo = (lo + Math.imul(al0, bl8)) | 0;
      mid = (mid + Math.imul(al0, bh8)) | 0;
      mid = (mid + Math.imul(ah0, bl8)) | 0;
      hi = (hi + Math.imul(ah0, bh8)) | 0;
      var w8 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w8 >>> 26)) | 0;
      w8 &= 0x3ffffff;
      lo = Math.imul(al9, bl0);
      mid = Math.imul(al9, bh0);
      mid = (mid + Math.imul(ah9, bl0)) | 0;
      hi = Math.imul(ah9, bh0);
      lo = (lo + Math.imul(al8, bl1)) | 0;
      mid = (mid + Math.imul(al8, bh1)) | 0;
      mid = (mid + Math.imul(ah8, bl1)) | 0;
      hi = (hi + Math.imul(ah8, bh1)) | 0;
      lo = (lo + Math.imul(al7, bl2)) | 0;
      mid = (mid + Math.imul(al7, bh2)) | 0;
      mid = (mid + Math.imul(ah7, bl2)) | 0;
      hi = (hi + Math.imul(ah7, bh2)) | 0;
      lo = (lo + Math.imul(al6, bl3)) | 0;
      mid = (mid + Math.imul(al6, bh3)) | 0;
      mid = (mid + Math.imul(ah6, bl3)) | 0;
      hi = (hi + Math.imul(ah6, bh3)) | 0;
      lo = (lo + Math.imul(al5, bl4)) | 0;
      mid = (mid + Math.imul(al5, bh4)) | 0;
      mid = (mid + Math.imul(ah5, bl4)) | 0;
      hi = (hi + Math.imul(ah5, bh4)) | 0;
      lo = (lo + Math.imul(al4, bl5)) | 0;
      mid = (mid + Math.imul(al4, bh5)) | 0;
      mid = (mid + Math.imul(ah4, bl5)) | 0;
      hi = (hi + Math.imul(ah4, bh5)) | 0;
      lo = (lo + Math.imul(al3, bl6)) | 0;
      mid = (mid + Math.imul(al3, bh6)) | 0;
      mid = (mid + Math.imul(ah3, bl6)) | 0;
      hi = (hi + Math.imul(ah3, bh6)) | 0;
      lo = (lo + Math.imul(al2, bl7)) | 0;
      mid = (mid + Math.imul(al2, bh7)) | 0;
      mid = (mid + Math.imul(ah2, bl7)) | 0;
      hi = (hi + Math.imul(ah2, bh7)) | 0;
      lo = (lo + Math.imul(al1, bl8)) | 0;
      mid = (mid + Math.imul(al1, bh8)) | 0;
      mid = (mid + Math.imul(ah1, bl8)) | 0;
      hi = (hi + Math.imul(ah1, bh8)) | 0;
      lo = (lo + Math.imul(al0, bl9)) | 0;
      mid = (mid + Math.imul(al0, bh9)) | 0;
      mid = (mid + Math.imul(ah0, bl9)) | 0;
      hi = (hi + Math.imul(ah0, bh9)) | 0;
      var w9 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w9 >>> 26)) | 0;
      w9 &= 0x3ffffff;
      lo = Math.imul(al9, bl1);
      mid = Math.imul(al9, bh1);
      mid = (mid + Math.imul(ah9, bl1)) | 0;
      hi = Math.imul(ah9, bh1);
      lo = (lo + Math.imul(al8, bl2)) | 0;
      mid = (mid + Math.imul(al8, bh2)) | 0;
      mid = (mid + Math.imul(ah8, bl2)) | 0;
      hi = (hi + Math.imul(ah8, bh2)) | 0;
      lo = (lo + Math.imul(al7, bl3)) | 0;
      mid = (mid + Math.imul(al7, bh3)) | 0;
      mid = (mid + Math.imul(ah7, bl3)) | 0;
      hi = (hi + Math.imul(ah7, bh3)) | 0;
      lo = (lo + Math.imul(al6, bl4)) | 0;
      mid = (mid + Math.imul(al6, bh4)) | 0;
      mid = (mid + Math.imul(ah6, bl4)) | 0;
      hi = (hi + Math.imul(ah6, bh4)) | 0;
      lo = (lo + Math.imul(al5, bl5)) | 0;
      mid = (mid + Math.imul(al5, bh5)) | 0;
      mid = (mid + Math.imul(ah5, bl5)) | 0;
      hi = (hi + Math.imul(ah5, bh5)) | 0;
      lo = (lo + Math.imul(al4, bl6)) | 0;
      mid = (mid + Math.imul(al4, bh6)) | 0;
      mid = (mid + Math.imul(ah4, bl6)) | 0;
      hi = (hi + Math.imul(ah4, bh6)) | 0;
      lo = (lo + Math.imul(al3, bl7)) | 0;
      mid = (mid + Math.imul(al3, bh7)) | 0;
      mid = (mid + Math.imul(ah3, bl7)) | 0;
      hi = (hi + Math.imul(ah3, bh7)) | 0;
      lo = (lo + Math.imul(al2, bl8)) | 0;
      mid = (mid + Math.imul(al2, bh8)) | 0;
      mid = (mid + Math.imul(ah2, bl8)) | 0;
      hi = (hi + Math.imul(ah2, bh8)) | 0;
      lo = (lo + Math.imul(al1, bl9)) | 0;
      mid = (mid + Math.imul(al1, bh9)) | 0;
      mid = (mid + Math.imul(ah1, bl9)) | 0;
      hi = (hi + Math.imul(ah1, bh9)) | 0;
      var w10 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w10 >>> 26)) | 0;
      w10 &= 0x3ffffff;
      lo = Math.imul(al9, bl2);
      mid = Math.imul(al9, bh2);
      mid = (mid + Math.imul(ah9, bl2)) | 0;
      hi = Math.imul(ah9, bh2);
      lo = (lo + Math.imul(al8, bl3)) | 0;
      mid = (mid + Math.imul(al8, bh3)) | 0;
      mid = (mid + Math.imul(ah8, bl3)) | 0;
      hi = (hi + Math.imul(ah8, bh3)) | 0;
      lo = (lo + Math.imul(al7, bl4)) | 0;
      mid = (mid + Math.imul(al7, bh4)) | 0;
      mid = (mid + Math.imul(ah7, bl4)) | 0;
      hi = (hi + Math.imul(ah7, bh4)) | 0;
      lo = (lo + Math.imul(al6, bl5)) | 0;
      mid = (mid + Math.imul(al6, bh5)) | 0;
      mid = (mid + Math.imul(ah6, bl5)) | 0;
      hi = (hi + Math.imul(ah6, bh5)) | 0;
      lo = (lo + Math.imul(al5, bl6)) | 0;
      mid = (mid + Math.imul(al5, bh6)) | 0;
      mid = (mid + Math.imul(ah5, bl6)) | 0;
      hi = (hi + Math.imul(ah5, bh6)) | 0;
      lo = (lo + Math.imul(al4, bl7)) | 0;
      mid = (mid + Math.imul(al4, bh7)) | 0;
      mid = (mid + Math.imul(ah4, bl7)) | 0;
      hi = (hi + Math.imul(ah4, bh7)) | 0;
      lo = (lo + Math.imul(al3, bl8)) | 0;
      mid = (mid + Math.imul(al3, bh8)) | 0;
      mid = (mid + Math.imul(ah3, bl8)) | 0;
      hi = (hi + Math.imul(ah3, bh8)) | 0;
      lo = (lo + Math.imul(al2, bl9)) | 0;
      mid = (mid + Math.imul(al2, bh9)) | 0;
      mid = (mid + Math.imul(ah2, bl9)) | 0;
      hi = (hi + Math.imul(ah2, bh9)) | 0;
      var w11 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w11 >>> 26)) | 0;
      w11 &= 0x3ffffff;
      lo = Math.imul(al9, bl3);
      mid = Math.imul(al9, bh3);
      mid = (mid + Math.imul(ah9, bl3)) | 0;
      hi = Math.imul(ah9, bh3);
      lo = (lo + Math.imul(al8, bl4)) | 0;
      mid = (mid + Math.imul(al8, bh4)) | 0;
      mid = (mid + Math.imul(ah8, bl4)) | 0;
      hi = (hi + Math.imul(ah8, bh4)) | 0;
      lo = (lo + Math.imul(al7, bl5)) | 0;
      mid = (mid + Math.imul(al7, bh5)) | 0;
      mid = (mid + Math.imul(ah7, bl5)) | 0;
      hi = (hi + Math.imul(ah7, bh5)) | 0;
      lo = (lo + Math.imul(al6, bl6)) | 0;
      mid = (mid + Math.imul(al6, bh6)) | 0;
      mid = (mid + Math.imul(ah6, bl6)) | 0;
      hi = (hi + Math.imul(ah6, bh6)) | 0;
      lo = (lo + Math.imul(al5, bl7)) | 0;
      mid = (mid + Math.imul(al5, bh7)) | 0;
      mid = (mid + Math.imul(ah5, bl7)) | 0;
      hi = (hi + Math.imul(ah5, bh7)) | 0;
      lo = (lo + Math.imul(al4, bl8)) | 0;
      mid = (mid + Math.imul(al4, bh8)) | 0;
      mid = (mid + Math.imul(ah4, bl8)) | 0;
      hi = (hi + Math.imul(ah4, bh8)) | 0;
      lo = (lo + Math.imul(al3, bl9)) | 0;
      mid = (mid + Math.imul(al3, bh9)) | 0;
      mid = (mid + Math.imul(ah3, bl9)) | 0;
      hi = (hi + Math.imul(ah3, bh9)) | 0;
      var w12 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w12 >>> 26)) | 0;
      w12 &= 0x3ffffff;
      lo = Math.imul(al9, bl4);
      mid = Math.imul(al9, bh4);
      mid = (mid + Math.imul(ah9, bl4)) | 0;
      hi = Math.imul(ah9, bh4);
      lo = (lo + Math.imul(al8, bl5)) | 0;
      mid = (mid + Math.imul(al8, bh5)) | 0;
      mid = (mid + Math.imul(ah8, bl5)) | 0;
      hi = (hi + Math.imul(ah8, bh5)) | 0;
      lo = (lo + Math.imul(al7, bl6)) | 0;
      mid = (mid + Math.imul(al7, bh6)) | 0;
      mid = (mid + Math.imul(ah7, bl6)) | 0;
      hi = (hi + Math.imul(ah7, bh6)) | 0;
      lo = (lo + Math.imul(al6, bl7)) | 0;
      mid = (mid + Math.imul(al6, bh7)) | 0;
      mid = (mid + Math.imul(ah6, bl7)) | 0;
      hi = (hi + Math.imul(ah6, bh7)) | 0;
      lo = (lo + Math.imul(al5, bl8)) | 0;
      mid = (mid + Math.imul(al5, bh8)) | 0;
      mid = (mid + Math.imul(ah5, bl8)) | 0;
      hi = (hi + Math.imul(ah5, bh8)) | 0;
      lo = (lo + Math.imul(al4, bl9)) | 0;
      mid = (mid + Math.imul(al4, bh9)) | 0;
      mid = (mid + Math.imul(ah4, bl9)) | 0;
      hi = (hi + Math.imul(ah4, bh9)) | 0;
      var w13 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w13 >>> 26)) | 0;
      w13 &= 0x3ffffff;
      lo = Math.imul(al9, bl5);
      mid = Math.imul(al9, bh5);
      mid = (mid + Math.imul(ah9, bl5)) | 0;
      hi = Math.imul(ah9, bh5);
      lo = (lo + Math.imul(al8, bl6)) | 0;
      mid = (mid + Math.imul(al8, bh6)) | 0;
      mid = (mid + Math.imul(ah8, bl6)) | 0;
      hi = (hi + Math.imul(ah8, bh6)) | 0;
      lo = (lo + Math.imul(al7, bl7)) | 0;
      mid = (mid + Math.imul(al7, bh7)) | 0;
      mid = (mid + Math.imul(ah7, bl7)) | 0;
      hi = (hi + Math.imul(ah7, bh7)) | 0;
      lo = (lo + Math.imul(al6, bl8)) | 0;
      mid = (mid + Math.imul(al6, bh8)) | 0;
      mid = (mid + Math.imul(ah6, bl8)) | 0;
      hi = (hi + Math.imul(ah6, bh8)) | 0;
      lo = (lo + Math.imul(al5, bl9)) | 0;
      mid = (mid + Math.imul(al5, bh9)) | 0;
      mid = (mid + Math.imul(ah5, bl9)) | 0;
      hi = (hi + Math.imul(ah5, bh9)) | 0;
      var w14 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w14 >>> 26)) | 0;
      w14 &= 0x3ffffff;
      lo = Math.imul(al9, bl6);
      mid = Math.imul(al9, bh6);
      mid = (mid + Math.imul(ah9, bl6)) | 0;
      hi = Math.imul(ah9, bh6);
      lo = (lo + Math.imul(al8, bl7)) | 0;
      mid = (mid + Math.imul(al8, bh7)) | 0;
      mid = (mid + Math.imul(ah8, bl7)) | 0;
      hi = (hi + Math.imul(ah8, bh7)) | 0;
      lo = (lo + Math.imul(al7, bl8)) | 0;
      mid = (mid + Math.imul(al7, bh8)) | 0;
      mid = (mid + Math.imul(ah7, bl8)) | 0;
      hi = (hi + Math.imul(ah7, bh8)) | 0;
      lo = (lo + Math.imul(al6, bl9)) | 0;
      mid = (mid + Math.imul(al6, bh9)) | 0;
      mid = (mid + Math.imul(ah6, bl9)) | 0;
      hi = (hi + Math.imul(ah6, bh9)) | 0;
      var w15 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w15 >>> 26)) | 0;
      w15 &= 0x3ffffff;
      lo = Math.imul(al9, bl7);
      mid = Math.imul(al9, bh7);
      mid = (mid + Math.imul(ah9, bl7)) | 0;
      hi = Math.imul(ah9, bh7);
      lo = (lo + Math.imul(al8, bl8)) | 0;
      mid = (mid + Math.imul(al8, bh8)) | 0;
      mid = (mid + Math.imul(ah8, bl8)) | 0;
      hi = (hi + Math.imul(ah8, bh8)) | 0;
      lo = (lo + Math.imul(al7, bl9)) | 0;
      mid = (mid + Math.imul(al7, bh9)) | 0;
      mid = (mid + Math.imul(ah7, bl9)) | 0;
      hi = (hi + Math.imul(ah7, bh9)) | 0;
      var w16 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w16 >>> 26)) | 0;
      w16 &= 0x3ffffff;
      lo = Math.imul(al9, bl8);
      mid = Math.imul(al9, bh8);
      mid = (mid + Math.imul(ah9, bl8)) | 0;
      hi = Math.imul(ah9, bh8);
      lo = (lo + Math.imul(al8, bl9)) | 0;
      mid = (mid + Math.imul(al8, bh9)) | 0;
      mid = (mid + Math.imul(ah8, bl9)) | 0;
      hi = (hi + Math.imul(ah8, bh9)) | 0;
      var w17 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w17 >>> 26)) | 0;
      w17 &= 0x3ffffff;
      lo = Math.imul(al9, bl9);
      mid = Math.imul(al9, bh9);
      mid = (mid + Math.imul(ah9, bl9)) | 0;
      hi = Math.imul(ah9, bh9);
      var w18 = (((c + lo) | 0) + ((mid & 0x1fff) << 13)) | 0;
      c = (((hi + (mid >>> 13)) | 0) + (w18 >>> 26)) | 0;
      w18 &= 0x3ffffff;
      o[0] = w0;
      o[1] = w1;
      o[2] = w2;
      o[3] = w3;
      o[4] = w4;
      o[5] = w5;
      o[6] = w6;
      o[7] = w7;
      o[8] = w8;
      o[9] = w9;
      o[10] = w10;
      o[11] = w11;
      o[12] = w12;
      o[13] = w13;
      o[14] = w14;
      o[15] = w15;
      o[16] = w16;
      o[17] = w17;
      o[18] = w18;
      if (c !== 0) {
        o[19] = c;
        out.length++;
      }
      return out;
    };
    if (!Math.imul) {
      comb10MulTo = smallMulTo;
    }
    function bigMulTo (self, num, out) {
      out.negative = num.negative ^ self.negative;
      out.length = self.length + num.length;
      var carry = 0;
      var hncarry = 0;
      for (var k = 0; k < out.length - 1; k++) {
        var ncarry = hncarry;
        hncarry = 0;
        var rword = carry & 0x3ffffff;
        var maxJ = Math.min(k, num.length - 1);
        for (var j = Math.max(0, k - self.length + 1); j <= maxJ; j++) {
          var i = k - j;
          var a = self.words[i] | 0;
          var b = num.words[j] | 0;
          var r = a * b;
          var lo = r & 0x3ffffff;
          ncarry = (ncarry + ((r / 0x4000000) | 0)) | 0;
          lo = (lo + rword) | 0;
          rword = lo & 0x3ffffff;
          ncarry = (ncarry + (lo >>> 26)) | 0;
          hncarry += ncarry >>> 26;
          ncarry &= 0x3ffffff;
        }
        out.words[k] = rword;
        carry = ncarry;
        ncarry = hncarry;
      }
      if (carry !== 0) {
        out.words[k] = carry;
      } else {
        out.length--;
      }
      return out._strip();
    }
    function jumboMulTo (self, num, out) {
      return bigMulTo(self, num, out);
    }
    BN.prototype.mulTo = function mulTo (num, out) {
      var res;
      var len = this.length + num.length;
      if (this.length === 10 && num.length === 10) {
        res = comb10MulTo(this, num, out);
      } else if (len < 63) {
        res = smallMulTo(this, num, out);
      } else if (len < 1024) {
        res = bigMulTo(this, num, out);
      } else {
        res = jumboMulTo(this, num, out);
      }
      return res;
    };
    BN.prototype.mul = function mul (num) {
      var out = new BN(null);
      out.words = new Array(this.length + num.length);
      return this.mulTo(num, out);
    };
    BN.prototype.mulf = function mulf (num) {
      var out = new BN(null);
      out.words = new Array(this.length + num.length);
      return jumboMulTo(this, num, out);
    };
    BN.prototype.imul = function imul (num) {
      return this.clone().mulTo(num, this);
    };
    BN.prototype.imuln = function imuln (num) {
      var isNegNum = num < 0;
      if (isNegNum) num = -num;
      assert(typeof num === 'number');
      assert(num < 0x4000000);
      var carry = 0;
      for (var i = 0; i < this.length; i++) {
        var w = (this.words[i] | 0) * num;
        var lo = (w & 0x3ffffff) + (carry & 0x3ffffff);
        carry >>= 26;
        carry += (w / 0x4000000) | 0;
        carry += lo >>> 26;
        this.words[i] = lo & 0x3ffffff;
      }
      if (carry !== 0) {
        this.words[i] = carry;
        this.length++;
      }
      return isNegNum ? this.ineg() : this;
    };
    BN.prototype.muln = function muln (num) {
      return this.clone().imuln(num);
    };
    BN.prototype.sqr = function sqr () {
      return this.mul(this);
    };
    BN.prototype.isqr = function isqr () {
      return this.imul(this.clone());
    };
    BN.prototype.pow = function pow (num) {
      var w = toBitArray(num);
      if (w.length === 0) return new BN(1);
      var res = this;
      for (var i = 0; i < w.length; i++, res = res.sqr()) {
        if (w[i] !== 0) break;
      }
      if (++i < w.length) {
        for (var q = res.sqr(); i < w.length; i++, q = q.sqr()) {
          if (w[i] === 0) continue;
          res = res.mul(q);
        }
      }
      return res;
    };
    BN.prototype.iushln = function iushln (bits) {
      assert(typeof bits === 'number' && bits >= 0);
      var r = bits % 26;
      var s = (bits - r) / 26;
      var carryMask = (0x3ffffff >>> (26 - r)) << (26 - r);
      var i;
      if (r !== 0) {
        var carry = 0;
        for (i = 0; i < this.length; i++) {
          var newCarry = this.words[i] & carryMask;
          var c = ((this.words[i] | 0) - newCarry) << r;
          this.words[i] = c | carry;
          carry = newCarry >>> (26 - r);
        }
        if (carry) {
          this.words[i] = carry;
          this.length++;
        }
      }
      if (s !== 0) {
        for (i = this.length - 1; i >= 0; i--) {
          this.words[i + s] = this.words[i];
        }
        for (i = 0; i < s; i++) {
          this.words[i] = 0;
        }
        this.length += s;
      }
      return this._strip();
    };
    BN.prototype.ishln = function ishln (bits) {
      assert(this.negative === 0);
      return this.iushln(bits);
    };
    BN.prototype.iushrn = function iushrn (bits, hint, extended) {
      assert(typeof bits === 'number' && bits >= 0);
      var h;
      if (hint) {
        h = (hint - (hint % 26)) / 26;
      } else {
        h = 0;
      }
      var r = bits % 26;
      var s = Math.min((bits - r) / 26, this.length);
      var mask = 0x3ffffff ^ ((0x3ffffff >>> r) << r);
      var maskedWords = extended;
      h -= s;
      h = Math.max(0, h);
      if (maskedWords) {
        for (var i = 0; i < s; i++) {
          maskedWords.words[i] = this.words[i];
        }
        maskedWords.length = s;
      }
      if (s === 0) ; else if (this.length > s) {
        this.length -= s;
        for (i = 0; i < this.length; i++) {
          this.words[i] = this.words[i + s];
        }
      } else {
        this.words[0] = 0;
        this.length = 1;
      }
      var carry = 0;
      for (i = this.length - 1; i >= 0 && (carry !== 0 || i >= h); i--) {
        var word = this.words[i] | 0;
        this.words[i] = (carry << (26 - r)) | (word >>> r);
        carry = word & mask;
      }
      if (maskedWords && carry !== 0) {
        maskedWords.words[maskedWords.length++] = carry;
      }
      if (this.length === 0) {
        this.words[0] = 0;
        this.length = 1;
      }
      return this._strip();
    };
    BN.prototype.ishrn = function ishrn (bits, hint, extended) {
      assert(this.negative === 0);
      return this.iushrn(bits, hint, extended);
    };
    BN.prototype.shln = function shln (bits) {
      return this.clone().ishln(bits);
    };
    BN.prototype.ushln = function ushln (bits) {
      return this.clone().iushln(bits);
    };
    BN.prototype.shrn = function shrn (bits) {
      return this.clone().ishrn(bits);
    };
    BN.prototype.ushrn = function ushrn (bits) {
      return this.clone().iushrn(bits);
    };
    BN.prototype.testn = function testn (bit) {
      assert(typeof bit === 'number' && bit >= 0);
      var r = bit % 26;
      var s = (bit - r) / 26;
      var q = 1 << r;
      if (this.length <= s) return false;
      var w = this.words[s];
      return !!(w & q);
    };
    BN.prototype.imaskn = function imaskn (bits) {
      assert(typeof bits === 'number' && bits >= 0);
      var r = bits % 26;
      var s = (bits - r) / 26;
      assert(this.negative === 0, 'imaskn works only with positive numbers');
      if (this.length <= s) {
        return this;
      }
      if (r !== 0) {
        s++;
      }
      this.length = Math.min(s, this.length);
      if (r !== 0) {
        var mask = 0x3ffffff ^ ((0x3ffffff >>> r) << r);
        this.words[this.length - 1] &= mask;
      }
      return this._strip();
    };
    BN.prototype.maskn = function maskn (bits) {
      return this.clone().imaskn(bits);
    };
    BN.prototype.iaddn = function iaddn (num) {
      assert(typeof num === 'number');
      assert(num < 0x4000000);
      if (num < 0) return this.isubn(-num);
      if (this.negative !== 0) {
        if (this.length === 1 && (this.words[0] | 0) <= num) {
          this.words[0] = num - (this.words[0] | 0);
          this.negative = 0;
          return this;
        }
        this.negative = 0;
        this.isubn(num);
        this.negative = 1;
        return this;
      }
      return this._iaddn(num);
    };
    BN.prototype._iaddn = function _iaddn (num) {
      this.words[0] += num;
      for (var i = 0; i < this.length && this.words[i] >= 0x4000000; i++) {
        this.words[i] -= 0x4000000;
        if (i === this.length - 1) {
          this.words[i + 1] = 1;
        } else {
          this.words[i + 1]++;
        }
      }
      this.length = Math.max(this.length, i + 1);
      return this;
    };
    BN.prototype.isubn = function isubn (num) {
      assert(typeof num === 'number');
      assert(num < 0x4000000);
      if (num < 0) return this.iaddn(-num);
      if (this.negative !== 0) {
        this.negative = 0;
        this.iaddn(num);
        this.negative = 1;
        return this;
      }
      this.words[0] -= num;
      if (this.length === 1 && this.words[0] < 0) {
        this.words[0] = -this.words[0];
        this.negative = 1;
      } else {
        for (var i = 0; i < this.length && this.words[i] < 0; i++) {
          this.words[i] += 0x4000000;
          this.words[i + 1] -= 1;
        }
      }
      return this._strip();
    };
    BN.prototype.addn = function addn (num) {
      return this.clone().iaddn(num);
    };
    BN.prototype.subn = function subn (num) {
      return this.clone().isubn(num);
    };
    BN.prototype.iabs = function iabs () {
      this.negative = 0;
      return this;
    };
    BN.prototype.abs = function abs () {
      return this.clone().iabs();
    };
    BN.prototype._ishlnsubmul = function _ishlnsubmul (num, mul, shift) {
      var len = num.length + shift;
      var i;
      this._expand(len);
      var w;
      var carry = 0;
      for (i = 0; i < num.length; i++) {
        w = (this.words[i + shift] | 0) + carry;
        var right = (num.words[i] | 0) * mul;
        w -= right & 0x3ffffff;
        carry = (w >> 26) - ((right / 0x4000000) | 0);
        this.words[i + shift] = w & 0x3ffffff;
      }
      for (; i < this.length - shift; i++) {
        w = (this.words[i + shift] | 0) + carry;
        carry = w >> 26;
        this.words[i + shift] = w & 0x3ffffff;
      }
      if (carry === 0) return this._strip();
      assert(carry === -1);
      carry = 0;
      for (i = 0; i < this.length; i++) {
        w = -(this.words[i] | 0) + carry;
        carry = w >> 26;
        this.words[i] = w & 0x3ffffff;
      }
      this.negative = 1;
      return this._strip();
    };
    BN.prototype._wordDiv = function _wordDiv (num, mode) {
      var shift = this.length - num.length;
      var a = this.clone();
      var b = num;
      var bhi = b.words[b.length - 1] | 0;
      var bhiBits = this._countBits(bhi);
      shift = 26 - bhiBits;
      if (shift !== 0) {
        b = b.ushln(shift);
        a.iushln(shift);
        bhi = b.words[b.length - 1] | 0;
      }
      var m = a.length - b.length;
      var q;
      if (mode !== 'mod') {
        q = new BN(null);
        q.length = m + 1;
        q.words = new Array(q.length);
        for (var i = 0; i < q.length; i++) {
          q.words[i] = 0;
        }
      }
      var diff = a.clone()._ishlnsubmul(b, 1, m);
      if (diff.negative === 0) {
        a = diff;
        if (q) {
          q.words[m] = 1;
        }
      }
      for (var j = m - 1; j >= 0; j--) {
        var qj = (a.words[b.length + j] | 0) * 0x4000000 +
          (a.words[b.length + j - 1] | 0);
        qj = Math.min((qj / bhi) | 0, 0x3ffffff);
        a._ishlnsubmul(b, qj, j);
        while (a.negative !== 0) {
          qj--;
          a.negative = 0;
          a._ishlnsubmul(b, 1, j);
          if (!a.isZero()) {
            a.negative ^= 1;
          }
        }
        if (q) {
          q.words[j] = qj;
        }
      }
      if (q) {
        q._strip();
      }
      a._strip();
      if (mode !== 'div' && shift !== 0) {
        a.iushrn(shift);
      }
      return {
        div: q || null,
        mod: a
      };
    };
    BN.prototype.divmod = function divmod (num, mode, positive) {
      assert(!num.isZero());
      if (this.isZero()) {
        return {
          div: new BN(0),
          mod: new BN(0)
        };
      }
      var div, mod, res;
      if (this.negative !== 0 && num.negative === 0) {
        res = this.neg().divmod(num, mode);
        if (mode !== 'mod') {
          div = res.div.neg();
        }
        if (mode !== 'div') {
          mod = res.mod.neg();
          if (positive && mod.negative !== 0) {
            mod.iadd(num);
          }
        }
        return {
          div: div,
          mod: mod
        };
      }
      if (this.negative === 0 && num.negative !== 0) {
        res = this.divmod(num.neg(), mode);
        if (mode !== 'mod') {
          div = res.div.neg();
        }
        return {
          div: div,
          mod: res.mod
        };
      }
      if ((this.negative & num.negative) !== 0) {
        res = this.neg().divmod(num.neg(), mode);
        if (mode !== 'div') {
          mod = res.mod.neg();
          if (positive && mod.negative !== 0) {
            mod.isub(num);
          }
        }
        return {
          div: res.div,
          mod: mod
        };
      }
      if (num.length > this.length || this.cmp(num) < 0) {
        return {
          div: new BN(0),
          mod: this
        };
      }
      if (num.length === 1) {
        if (mode === 'div') {
          return {
            div: this.divn(num.words[0]),
            mod: null
          };
        }
        if (mode === 'mod') {
          return {
            div: null,
            mod: new BN(this.modrn(num.words[0]))
          };
        }
        return {
          div: this.divn(num.words[0]),
          mod: new BN(this.modrn(num.words[0]))
        };
      }
      return this._wordDiv(num, mode);
    };
    BN.prototype.div = function div (num) {
      return this.divmod(num, 'div', false).div;
    };
    BN.prototype.mod = function mod (num) {
      return this.divmod(num, 'mod', false).mod;
    };
    BN.prototype.umod = function umod (num) {
      return this.divmod(num, 'mod', true).mod;
    };
    BN.prototype.divRound = function divRound (num) {
      var dm = this.divmod(num);
      if (dm.mod.isZero()) return dm.div;
      var mod = dm.div.negative !== 0 ? dm.mod.isub(num) : dm.mod;
      var half = num.ushrn(1);
      var r2 = num.andln(1);
      var cmp = mod.cmp(half);
      if (cmp < 0 || (r2 === 1 && cmp === 0)) return dm.div;
      return dm.div.negative !== 0 ? dm.div.isubn(1) : dm.div.iaddn(1);
    };
    BN.prototype.modrn = function modrn (num) {
      var isNegNum = num < 0;
      if (isNegNum) num = -num;
      assert(num <= 0x3ffffff);
      var p = (1 << 26) % num;
      var acc = 0;
      for (var i = this.length - 1; i >= 0; i--) {
        acc = (p * acc + (this.words[i] | 0)) % num;
      }
      return isNegNum ? -acc : acc;
    };
    BN.prototype.modn = function modn (num) {
      return this.modrn(num);
    };
    BN.prototype.idivn = function idivn (num) {
      var isNegNum = num < 0;
      if (isNegNum) num = -num;
      assert(num <= 0x3ffffff);
      var carry = 0;
      for (var i = this.length - 1; i >= 0; i--) {
        var w = (this.words[i] | 0) + carry * 0x4000000;
        this.words[i] = (w / num) | 0;
        carry = w % num;
      }
      this._strip();
      return isNegNum ? this.ineg() : this;
    };
    BN.prototype.divn = function divn (num) {
      return this.clone().idivn(num);
    };
    BN.prototype.egcd = function egcd (p) {
      assert(p.negative === 0);
      assert(!p.isZero());
      var x = this;
      var y = p.clone();
      if (x.negative !== 0) {
        x = x.umod(p);
      } else {
        x = x.clone();
      }
      var A = new BN(1);
      var B = new BN(0);
      var C = new BN(0);
      var D = new BN(1);
      var g = 0;
      while (x.isEven() && y.isEven()) {
        x.iushrn(1);
        y.iushrn(1);
        ++g;
      }
      var yp = y.clone();
      var xp = x.clone();
      while (!x.isZero()) {
        for (var i = 0, im = 1; (x.words[0] & im) === 0 && i < 26; ++i, im <<= 1);
        if (i > 0) {
          x.iushrn(i);
          while (i-- > 0) {
            if (A.isOdd() || B.isOdd()) {
              A.iadd(yp);
              B.isub(xp);
            }
            A.iushrn(1);
            B.iushrn(1);
          }
        }
        for (var j = 0, jm = 1; (y.words[0] & jm) === 0 && j < 26; ++j, jm <<= 1);
        if (j > 0) {
          y.iushrn(j);
          while (j-- > 0) {
            if (C.isOdd() || D.isOdd()) {
              C.iadd(yp);
              D.isub(xp);
            }
            C.iushrn(1);
            D.iushrn(1);
          }
        }
        if (x.cmp(y) >= 0) {
          x.isub(y);
          A.isub(C);
          B.isub(D);
        } else {
          y.isub(x);
          C.isub(A);
          D.isub(B);
        }
      }
      return {
        a: C,
        b: D,
        gcd: y.iushln(g)
      };
    };
    BN.prototype._invmp = function _invmp (p) {
      assert(p.negative === 0);
      assert(!p.isZero());
      var a = this;
      var b = p.clone();
      if (a.negative !== 0) {
        a = a.umod(p);
      } else {
        a = a.clone();
      }
      var x1 = new BN(1);
      var x2 = new BN(0);
      var delta = b.clone();
      while (a.cmpn(1) > 0 && b.cmpn(1) > 0) {
        for (var i = 0, im = 1; (a.words[0] & im) === 0 && i < 26; ++i, im <<= 1);
        if (i > 0) {
          a.iushrn(i);
          while (i-- > 0) {
            if (x1.isOdd()) {
              x1.iadd(delta);
            }
            x1.iushrn(1);
          }
        }
        for (var j = 0, jm = 1; (b.words[0] & jm) === 0 && j < 26; ++j, jm <<= 1);
        if (j > 0) {
          b.iushrn(j);
          while (j-- > 0) {
            if (x2.isOdd()) {
              x2.iadd(delta);
            }
            x2.iushrn(1);
          }
        }
        if (a.cmp(b) >= 0) {
          a.isub(b);
          x1.isub(x2);
        } else {
          b.isub(a);
          x2.isub(x1);
        }
      }
      var res;
      if (a.cmpn(1) === 0) {
        res = x1;
      } else {
        res = x2;
      }
      if (res.cmpn(0) < 0) {
        res.iadd(p);
      }
      return res;
    };
    BN.prototype.gcd = function gcd (num) {
      if (this.isZero()) return num.abs();
      if (num.isZero()) return this.abs();
      var a = this.clone();
      var b = num.clone();
      a.negative = 0;
      b.negative = 0;
      for (var shift = 0; a.isEven() && b.isEven(); shift++) {
        a.iushrn(1);
        b.iushrn(1);
      }
      do {
        while (a.isEven()) {
          a.iushrn(1);
        }
        while (b.isEven()) {
          b.iushrn(1);
        }
        var r = a.cmp(b);
        if (r < 0) {
          var t = a;
          a = b;
          b = t;
        } else if (r === 0 || b.cmpn(1) === 0) {
          break;
        }
        a.isub(b);
      } while (true);
      return b.iushln(shift);
    };
    BN.prototype.invm = function invm (num) {
      return this.egcd(num).a.umod(num);
    };
    BN.prototype.isEven = function isEven () {
      return (this.words[0] & 1) === 0;
    };
    BN.prototype.isOdd = function isOdd () {
      return (this.words[0] & 1) === 1;
    };
    BN.prototype.andln = function andln (num) {
      return this.words[0] & num;
    };
    BN.prototype.bincn = function bincn (bit) {
      assert(typeof bit === 'number');
      var r = bit % 26;
      var s = (bit - r) / 26;
      var q = 1 << r;
      if (this.length <= s) {
        this._expand(s + 1);
        this.words[s] |= q;
        return this;
      }
      var carry = q;
      for (var i = s; carry !== 0 && i < this.length; i++) {
        var w = this.words[i] | 0;
        w += carry;
        carry = w >>> 26;
        w &= 0x3ffffff;
        this.words[i] = w;
      }
      if (carry !== 0) {
        this.words[i] = carry;
        this.length++;
      }
      return this;
    };
    BN.prototype.isZero = function isZero () {
      return this.length === 1 && this.words[0] === 0;
    };
    BN.prototype.cmpn = function cmpn (num) {
      var negative = num < 0;
      if (this.negative !== 0 && !negative) return -1;
      if (this.negative === 0 && negative) return 1;
      this._strip();
      var res;
      if (this.length > 1) {
        res = 1;
      } else {
        if (negative) {
          num = -num;
        }
        assert(num <= 0x3ffffff, 'Number is too big');
        var w = this.words[0] | 0;
        res = w === num ? 0 : w < num ? -1 : 1;
      }
      if (this.negative !== 0) return -res | 0;
      return res;
    };
    BN.prototype.cmp = function cmp (num) {
      if (this.negative !== 0 && num.negative === 0) return -1;
      if (this.negative === 0 && num.negative !== 0) return 1;
      var res = this.ucmp(num);
      if (this.negative !== 0) return -res | 0;
      return res;
    };
    BN.prototype.ucmp = function ucmp (num) {
      if (this.length > num.length) return 1;
      if (this.length < num.length) return -1;
      var res = 0;
      for (var i = this.length - 1; i >= 0; i--) {
        var a = this.words[i] | 0;
        var b = num.words[i] | 0;
        if (a === b) continue;
        if (a < b) {
          res = -1;
        } else if (a > b) {
          res = 1;
        }
        break;
      }
      return res;
    };
    BN.prototype.gtn = function gtn (num) {
      return this.cmpn(num) === 1;
    };
    BN.prototype.gt = function gt (num) {
      return this.cmp(num) === 1;
    };
    BN.prototype.gten = function gten (num) {
      return this.cmpn(num) >= 0;
    };
    BN.prototype.gte = function gte (num) {
      return this.cmp(num) >= 0;
    };
    BN.prototype.ltn = function ltn (num) {
      return this.cmpn(num) === -1;
    };
    BN.prototype.lt = function lt (num) {
      return this.cmp(num) === -1;
    };
    BN.prototype.lten = function lten (num) {
      return this.cmpn(num) <= 0;
    };
    BN.prototype.lte = function lte (num) {
      return this.cmp(num) <= 0;
    };
    BN.prototype.eqn = function eqn (num) {
      return this.cmpn(num) === 0;
    };
    BN.prototype.eq = function eq (num) {
      return this.cmp(num) === 0;
    };
    BN.red = function red (num) {
      return new Red(num);
    };
    BN.prototype.toRed = function toRed (ctx) {
      assert(!this.red, 'Already a number in reduction context');
      assert(this.negative === 0, 'red works only with positives');
      return ctx.convertTo(this)._forceRed(ctx);
    };
    BN.prototype.fromRed = function fromRed () {
      assert(this.red, 'fromRed works only with numbers in reduction context');
      return this.red.convertFrom(this);
    };
    BN.prototype._forceRed = function _forceRed (ctx) {
      this.red = ctx;
      return this;
    };
    BN.prototype.forceRed = function forceRed (ctx) {
      assert(!this.red, 'Already a number in reduction context');
      return this._forceRed(ctx);
    };
    BN.prototype.redAdd = function redAdd (num) {
      assert(this.red, 'redAdd works only with red numbers');
      return this.red.add(this, num);
    };
    BN.prototype.redIAdd = function redIAdd (num) {
      assert(this.red, 'redIAdd works only with red numbers');
      return this.red.iadd(this, num);
    };
    BN.prototype.redSub = function redSub (num) {
      assert(this.red, 'redSub works only with red numbers');
      return this.red.sub(this, num);
    };
    BN.prototype.redISub = function redISub (num) {
      assert(this.red, 'redISub works only with red numbers');
      return this.red.isub(this, num);
    };
    BN.prototype.redShl = function redShl (num) {
      assert(this.red, 'redShl works only with red numbers');
      return this.red.shl(this, num);
    };
    BN.prototype.redMul = function redMul (num) {
      assert(this.red, 'redMul works only with red numbers');
      this.red._verify2(this, num);
      return this.red.mul(this, num);
    };
    BN.prototype.redIMul = function redIMul (num) {
      assert(this.red, 'redMul works only with red numbers');
      this.red._verify2(this, num);
      return this.red.imul(this, num);
    };
    BN.prototype.redSqr = function redSqr () {
      assert(this.red, 'redSqr works only with red numbers');
      this.red._verify1(this);
      return this.red.sqr(this);
    };
    BN.prototype.redISqr = function redISqr () {
      assert(this.red, 'redISqr works only with red numbers');
      this.red._verify1(this);
      return this.red.isqr(this);
    };
    BN.prototype.redSqrt = function redSqrt () {
      assert(this.red, 'redSqrt works only with red numbers');
      this.red._verify1(this);
      return this.red.sqrt(this);
    };
    BN.prototype.redInvm = function redInvm () {
      assert(this.red, 'redInvm works only with red numbers');
      this.red._verify1(this);
      return this.red.invm(this);
    };
    BN.prototype.redNeg = function redNeg () {
      assert(this.red, 'redNeg works only with red numbers');
      this.red._verify1(this);
      return this.red.neg(this);
    };
    BN.prototype.redPow = function redPow (num) {
      assert(this.red && !num.red, 'redPow(normalNum)');
      this.red._verify1(this);
      return this.red.pow(this, num);
    };
    var primes = {
      k256: null,
      p224: null,
      p192: null,
      p25519: null
    };
    function MPrime (name, p) {
      this.name = name;
      this.p = new BN(p, 16);
      this.n = this.p.bitLength();
      this.k = new BN(1).iushln(this.n).isub(this.p);
      this.tmp = this._tmp();
    }
    MPrime.prototype._tmp = function _tmp () {
      var tmp = new BN(null);
      tmp.words = new Array(Math.ceil(this.n / 13));
      return tmp;
    };
    MPrime.prototype.ireduce = function ireduce (num) {
      var r = num;
      var rlen;
      do {
        this.split(r, this.tmp);
        r = this.imulK(r);
        r = r.iadd(this.tmp);
        rlen = r.bitLength();
      } while (rlen > this.n);
      var cmp = rlen < this.n ? -1 : r.ucmp(this.p);
      if (cmp === 0) {
        r.words[0] = 0;
        r.length = 1;
      } else if (cmp > 0) {
        r.isub(this.p);
      } else {
        if (r.strip !== undefined) {
          r.strip();
        } else {
          r._strip();
        }
      }
      return r;
    };
    MPrime.prototype.split = function split (input, out) {
      input.iushrn(this.n, 0, out);
    };
    MPrime.prototype.imulK = function imulK (num) {
      return num.imul(this.k);
    };
    function K256 () {
      MPrime.call(
        this,
        'k256',
        'ffffffff ffffffff ffffffff ffffffff ffffffff ffffffff fffffffe fffffc2f');
    }
    inherits(K256, MPrime);
    K256.prototype.split = function split (input, output) {
      var mask = 0x3fffff;
      var outLen = Math.min(input.length, 9);
      for (var i = 0; i < outLen; i++) {
        output.words[i] = input.words[i];
      }
      output.length = outLen;
      if (input.length <= 9) {
        input.words[0] = 0;
        input.length = 1;
        return;
      }
      var prev = input.words[9];
      output.words[output.length++] = prev & mask;
      for (i = 10; i < input.length; i++) {
        var next = input.words[i] | 0;
        input.words[i - 10] = ((next & mask) << 4) | (prev >>> 22);
        prev = next;
      }
      prev >>>= 22;
      input.words[i - 10] = prev;
      if (prev === 0 && input.length > 10) {
        input.length -= 10;
      } else {
        input.length -= 9;
      }
    };
    K256.prototype.imulK = function imulK (num) {
      num.words[num.length] = 0;
      num.words[num.length + 1] = 0;
      num.length += 2;
      var lo = 0;
      for (var i = 0; i < num.length; i++) {
        var w = num.words[i] | 0;
        lo += w * 0x3d1;
        num.words[i] = lo & 0x3ffffff;
        lo = w * 0x40 + ((lo / 0x4000000) | 0);
      }
      if (num.words[num.length - 1] === 0) {
        num.length--;
        if (num.words[num.length - 1] === 0) {
          num.length--;
        }
      }
      return num;
    };
    function P224 () {
      MPrime.call(
        this,
        'p224',
        'ffffffff ffffffff ffffffff ffffffff 00000000 00000000 00000001');
    }
    inherits(P224, MPrime);
    function P192 () {
      MPrime.call(
        this,
        'p192',
        'ffffffff ffffffff ffffffff fffffffe ffffffff ffffffff');
    }
    inherits(P192, MPrime);
    function P25519 () {
      MPrime.call(
        this,
        '25519',
        '7fffffffffffffff ffffffffffffffff ffffffffffffffff ffffffffffffffed');
    }
    inherits(P25519, MPrime);
    P25519.prototype.imulK = function imulK (num) {
      var carry = 0;
      for (var i = 0; i < num.length; i++) {
        var hi = (num.words[i] | 0) * 0x13 + carry;
        var lo = hi & 0x3ffffff;
        hi >>>= 26;
        num.words[i] = lo;
        carry = hi;
      }
      if (carry !== 0) {
        num.words[num.length++] = carry;
      }
      return num;
    };
    BN._prime = function prime (name) {
      if (primes[name]) return primes[name];
      var prime;
      if (name === 'k256') {
        prime = new K256();
      } else if (name === 'p224') {
        prime = new P224();
      } else if (name === 'p192') {
        prime = new P192();
      } else if (name === 'p25519') {
        prime = new P25519();
      } else {
        throw new Error('Unknown prime ' + name);
      }
      primes[name] = prime;
      return prime;
    };
    function Red (m) {
      if (typeof m === 'string') {
        var prime = BN._prime(m);
        this.m = prime.p;
        this.prime = prime;
      } else {
        assert(m.gtn(1), 'modulus must be greater than 1');
        this.m = m;
        this.prime = null;
      }
    }
    Red.prototype._verify1 = function _verify1 (a) {
      assert(a.negative === 0, 'red works only with positives');
      assert(a.red, 'red works only with red numbers');
    };
    Red.prototype._verify2 = function _verify2 (a, b) {
      assert((a.negative | b.negative) === 0, 'red works only with positives');
      assert(a.red && a.red === b.red,
        'red works only with red numbers');
    };
    Red.prototype.imod = function imod (a) {
      if (this.prime) return this.prime.ireduce(a)._forceRed(this);
      move(a, a.umod(this.m)._forceRed(this));
      return a;
    };
    Red.prototype.neg = function neg (a) {
      if (a.isZero()) {
        return a.clone();
      }
      return this.m.sub(a)._forceRed(this);
    };
    Red.prototype.add = function add (a, b) {
      this._verify2(a, b);
      var res = a.add(b);
      if (res.cmp(this.m) >= 0) {
        res.isub(this.m);
      }
      return res._forceRed(this);
    };
    Red.prototype.iadd = function iadd (a, b) {
      this._verify2(a, b);
      var res = a.iadd(b);
      if (res.cmp(this.m) >= 0) {
        res.isub(this.m);
      }
      return res;
    };
    Red.prototype.sub = function sub (a, b) {
      this._verify2(a, b);
      var res = a.sub(b);
      if (res.cmpn(0) < 0) {
        res.iadd(this.m);
      }
      return res._forceRed(this);
    };
    Red.prototype.isub = function isub (a, b) {
      this._verify2(a, b);
      var res = a.isub(b);
      if (res.cmpn(0) < 0) {
        res.iadd(this.m);
      }
      return res;
    };
    Red.prototype.shl = function shl (a, num) {
      this._verify1(a);
      return this.imod(a.ushln(num));
    };
    Red.prototype.imul = function imul (a, b) {
      this._verify2(a, b);
      return this.imod(a.imul(b));
    };
    Red.prototype.mul = function mul (a, b) {
      this._verify2(a, b);
      return this.imod(a.mul(b));
    };
    Red.prototype.isqr = function isqr (a) {
      return this.imul(a, a.clone());
    };
    Red.prototype.sqr = function sqr (a) {
      return this.mul(a, a);
    };
    Red.prototype.sqrt = function sqrt (a) {
      if (a.isZero()) return a.clone();
      var mod3 = this.m.andln(3);
      assert(mod3 % 2 === 1);
      if (mod3 === 3) {
        var pow = this.m.add(new BN(1)).iushrn(2);
        return this.pow(a, pow);
      }
      var q = this.m.subn(1);
      var s = 0;
      while (!q.isZero() && q.andln(1) === 0) {
        s++;
        q.iushrn(1);
      }
      assert(!q.isZero());
      var one = new BN(1).toRed(this);
      var nOne = one.redNeg();
      var lpow = this.m.subn(1).iushrn(1);
      var z = this.m.bitLength();
      z = new BN(2 * z * z).toRed(this);
      while (this.pow(z, lpow).cmp(nOne) !== 0) {
        z.redIAdd(nOne);
      }
      var c = this.pow(z, q);
      var r = this.pow(a, q.addn(1).iushrn(1));
      var t = this.pow(a, q);
      var m = s;
      while (t.cmp(one) !== 0) {
        var tmp = t;
        for (var i = 0; tmp.cmp(one) !== 0; i++) {
          tmp = tmp.redSqr();
        }
        assert(i < m);
        var b = this.pow(c, new BN(1).iushln(m - i - 1));
        r = r.redMul(b);
        c = b.redSqr();
        t = t.redMul(c);
        m = i;
      }
      return r;
    };
    Red.prototype.invm = function invm (a) {
      var inv = a._invmp(this.m);
      if (inv.negative !== 0) {
        inv.negative = 0;
        return this.imod(inv).redNeg();
      } else {
        return this.imod(inv);
      }
    };
    Red.prototype.pow = function pow (a, num) {
      if (num.isZero()) return new BN(1).toRed(this);
      if (num.cmpn(1) === 0) return a.clone();
      var windowSize = 4;
      var wnd = new Array(1 << windowSize);
      wnd[0] = new BN(1).toRed(this);
      wnd[1] = a;
      for (var i = 2; i < wnd.length; i++) {
        wnd[i] = this.mul(wnd[i - 1], a);
      }
      var res = wnd[0];
      var current = 0;
      var currentLen = 0;
      var start = num.bitLength() % 26;
      if (start === 0) {
        start = 26;
      }
      for (i = num.length - 1; i >= 0; i--) {
        var word = num.words[i];
        for (var j = start - 1; j >= 0; j--) {
          var bit = (word >> j) & 1;
          if (res !== wnd[0]) {
            res = this.sqr(res);
          }
          if (bit === 0 && current === 0) {
            currentLen = 0;
            continue;
          }
          current <<= 1;
          current |= bit;
          currentLen++;
          if (currentLen !== windowSize && (i !== 0 || j !== 0)) continue;
          res = this.mul(res, wnd[current]);
          currentLen = 0;
          current = 0;
        }
        start = 26;
      }
      return res;
    };
    Red.prototype.convertTo = function convertTo (num) {
      var r = num.umod(this.m);
      return r === num ? r.clone() : r;
    };
    Red.prototype.convertFrom = function convertFrom (num) {
      var res = num.clone();
      res.red = null;
      return res;
    };
    BN.mont = function mont (num) {
      return new Mont(num);
    };
    function Mont (m) {
      Red.call(this, m);
      this.shift = this.m.bitLength();
      if (this.shift % 26 !== 0) {
        this.shift += 26 - (this.shift % 26);
      }
      this.r = new BN(1).iushln(this.shift);
      this.r2 = this.imod(this.r.sqr());
      this.rinv = this.r._invmp(this.m);
      this.minv = this.rinv.mul(this.r).isubn(1).div(this.m);
      this.minv = this.minv.umod(this.r);
      this.minv = this.r.sub(this.minv);
    }
    inherits(Mont, Red);
    Mont.prototype.convertTo = function convertTo (num) {
      return this.imod(num.ushln(this.shift));
    };
    Mont.prototype.convertFrom = function convertFrom (num) {
      var r = this.imod(num.mul(this.rinv));
      r.red = null;
      return r;
    };
    Mont.prototype.imul = function imul (a, b) {
      if (a.isZero() || b.isZero()) {
        a.words[0] = 0;
        a.length = 1;
        return a;
      }
      var t = a.imul(b);
      var c = t.maskn(this.shift).mul(this.minv).imaskn(this.shift).mul(this.m);
      var u = t.isub(c).iushrn(this.shift);
      var res = u;
      if (u.cmp(this.m) >= 0) {
        res = u.isub(this.m);
      } else if (u.cmpn(0) < 0) {
        res = u.iadd(this.m);
      }
      return res._forceRed(this);
    };
    Mont.prototype.mul = function mul (a, b) {
      if (a.isZero() || b.isZero()) return new BN(0)._forceRed(this);
      var t = a.mul(b);
      var c = t.maskn(this.shift).mul(this.minv).imaskn(this.shift).mul(this.m);
      var u = t.isub(c).iushrn(this.shift);
      var res = u;
      if (u.cmp(this.m) >= 0) {
        res = u.isub(this.m);
      } else if (u.cmpn(0) < 0) {
        res = u.iadd(this.m);
      }
      return res._forceRed(this);
    };
    Mont.prototype.invm = function invm (a) {
      var res = this.imod(a._invmp(this.m).mul(this.r2));
      return res._forceRed(this);
    };
  })(module, commonjsGlobal);
  }(bn));
  const BN = bn.exports;

  function isBn(value) {
    return BN.isBN(value);
  }

  function isObject(value) {
    return !!value && typeof value === 'object';
  }

  function isOn(...fns) {
    return value => (isObject(value) || isFunction(value)) && fns.every(f => isFunction(value[f]));
  }
  function isOnObject(...fns) {
    return value => isObject(value) && fns.every(f => isFunction(value[f]));
  }

  const isToBigInt = isOn('toBigInt');

  const isToBn = isOn('toBn');

  function nToBigInt(value) {
    return typeof value === 'bigint' ? value : !value ? BigInt(0) : isHex(value) ? hexToBigInt(value.toString()) : isBn(value) ? BigInt(value.toString()) : isToBigInt(value) ? value.toBigInt() : isToBn(value) ? BigInt(value.toBn().toString()) : BigInt(value);
  }

  const _sqrt2pow53n = BigInt(94906265);
  function nSqrt(value) {
    const n = nToBigInt(value);
    assert(n >= _0n, 'square root of negative numbers is not supported');
    if (n <= _2pow53n) {
      return BigInt(Math.floor(Math.sqrt(Number(n))));
    }
    let x0 = _sqrt2pow53n;
    while (true) {
      const x1 = n / x0 + x0 >> _1n;
      if (x0 === x1 || x0 === x1 - _1n) {
        return x0;
      }
      x0 = x1;
    }
  }

  function gt(a, b) {
    return a > b;
  }
  function lt(a, b) {
    return a < b;
  }
  function find$1(items, cmp) {
    assert(items.length >= 1, 'Must provide one or more bigint arguments');
    let result = items[0];
    for (let i = 1; i < items.length; i++) {
      if (cmp(items[i], result)) {
        result = items[i];
      }
    }
    return result;
  }
  function nMax(...items) {
    return find$1(items, gt);
  }
  function nMin(...items) {
    return find$1(items, lt);
  }

  const hasBigInt = typeof BigInt === 'function' && typeof BigInt.asIntN === 'function';
  const hasBuffer = typeof Buffer !== 'undefined';
  const hasCjs = typeof require === 'function' && typeof module !== 'undefined';
  const hasDirname = typeof __dirname !== 'undefined';
  const hasEsm = !hasCjs;
  const hasProcess = typeof process === 'object';
  const hasWasm = typeof WebAssembly !== 'undefined';

  function isBuffer(value) {
    return hasBuffer && Buffer.isBuffer(value);
  }

  function isU8a(value) {
    return value instanceof Uint8Array;
  }

  class TextEncoder$1 {
    encode(value) {
      const u8a = new Uint8Array(value.length);
      for (let i = 0; i < value.length; i++) {
        u8a[i] = value.charCodeAt(i);
      }
      return u8a;
    }
  }

  const TextEncoder = extractGlobal('TextEncoder', TextEncoder$1);

  const encoder = new TextEncoder();
  function stringToU8a(value) {
    return value ? encoder.encode(value.toString()) : new Uint8Array();
  }

  function u8aToU8a(value) {
    return !value ? new Uint8Array() : Array.isArray(value) || isBuffer(value) ? new Uint8Array(value) : isU8a(value) ? value : isHex(value) ? hexToU8a(value) : stringToU8a(value);
  }

  function u8aCmp(a, b) {
    const u8aa = u8aToU8a(a);
    const u8ab = u8aToU8a(b);
    let i = 0;
    while (true) {
      const overA = i >= u8aa.length;
      const overB = i >= u8ab.length;
      if (overA && overB) {
        return 0;
      } else if (overA) {
        return -1;
      } else if (overB) {
        return 1;
      } else if (u8aa[i] !== u8ab[i]) {
        return u8aa[i] > u8ab[i] ? 1 : -1;
      }
      i++;
    }
  }

  function u8aConcat(...list) {
    let length = 0;
    let offset = 0;
    const u8as = new Array(list.length);
    for (let i = 0; i < list.length; i++) {
      u8as[i] = u8aToU8a(list[i]);
      length += u8as[i].length;
    }
    const result = new Uint8Array(length);
    for (let i = 0; i < u8as.length; i++) {
      result.set(u8as[i], offset);
      offset += u8as[i].length;
    }
    return result;
  }

  function u8aEmpty(value) {
    for (let i = 0; i < value.length; i++) {
      if (value[i]) {
        return false;
      }
    }
    return true;
  }

  function u8aEq(a, b) {
    const u8aa = u8aToU8a(a);
    const u8ab = u8aToU8a(b);
    if (u8aa.length === u8ab.length) {
      const dvA = new DataView(u8aa.buffer, u8aa.byteOffset);
      const dvB = new DataView(u8ab.buffer, u8ab.byteOffset);
      const mod = u8aa.length % 4;
      const length = u8aa.length - mod;
      for (let i = 0; i < length; i += 4) {
        if (dvA.getUint32(i) !== dvB.getUint32(i)) {
          return false;
        }
      }
      for (let i = length; i < u8aa.length; i++) {
        if (u8aa[i] !== u8ab[i]) {
          return false;
        }
      }
      return true;
    }
    return false;
  }

  function u8aFixLength(value, bitLength = -1, atStart = false) {
    const byteLength = Math.ceil(bitLength / 8);
    if (bitLength === -1 || value.length === byteLength) {
      return value;
    } else if (value.length > byteLength) {
      return value.subarray(0, byteLength);
    }
    const result = new Uint8Array(byteLength);
    result.set(value, atStart ? 0 : byteLength - value.length);
    return result;
  }

  function u8aSorted(u8as) {
    return u8as.sort(u8aCmp);
  }

  function isBoolean(value) {
    return typeof value === 'boolean';
  }

  const DEFAULT_OPTS$3 = {
    isLe: false,
    isNegative: false
  };
  function hexToBn(value, options = DEFAULT_OPTS$3) {
    if (!value || value === '0x') {
      return new BN(0);
    }
    const {
      isLe,
      isNegative
    } = objectSpread({
      isLe: false,
      isNegative: false
    }, isBoolean(options) ? {
      isLe: options
    } : options);
    const stripped = hexStripPrefix(value);
    const bn = new BN(stripped, 16, isLe ? 'le' : 'be');
    return isNegative ? bn.fromTwos(stripped.length * 4) : bn;
  }

  function hex(value) {
    const mod = value.length % 2;
    const length = value.length - mod;
    const dv = new DataView(value.buffer, value.byteOffset);
    let result = '';
    for (let i = 0; i < length; i += 2) {
      result += U16_TO_HEX[dv.getUint16(i)];
    }
    if (mod) {
      result += U8_TO_HEX[dv.getUint8(length)];
    }
    return result;
  }
  function u8aToHex(value, bitLength = -1, isPrefixed = true) {
    const length = Math.ceil(bitLength / 8);
    return `${isPrefixed ? '0x' : ''}${!value || !value.length ? '' : length > 0 && value.length > length ? `${hex(value.subarray(0, length / 2))}…${hex(value.subarray(value.length - length / 2))}` : hex(value)}`;
  }

  const DEFAULT_OPTS$2 = {
    isLe: true,
    isNegative: false
  };
  function u8aToBn(value, options = DEFAULT_OPTS$2) {
    return hexToBn(u8aToHex(value), options);
  }

  function u8aToBuffer(value) {
    return Buffer.from(value || []);
  }

  class TextDecoder$1 {
    constructor(_) {
    }
    decode(value) {
      let result = '';
      for (let i = 0; i < value.length; i++) {
        result += String.fromCharCode(value[i]);
      }
      return result;
    }
  }

  const TextDecoder = extractGlobal('TextDecoder', TextDecoder$1);

  const decoder = new TextDecoder('utf-8');
  function u8aToString(value) {
    return !(value !== null && value !== void 0 && value.length) ? '' : decoder.decode(value);
  }

  const U8A_WRAP_ETHEREUM = u8aToU8a('\x19Ethereum Signed Message:\n');
  const U8A_WRAP_PREFIX = u8aToU8a('<Bytes>');
  const U8A_WRAP_POSTFIX = u8aToU8a('</Bytes>');
  const WRAP_LEN = U8A_WRAP_PREFIX.length + U8A_WRAP_POSTFIX.length;
  function u8aIsWrapped(u8a, withEthereum) {
    return u8a.length >= WRAP_LEN && u8aEq(u8a.subarray(0, U8A_WRAP_PREFIX.length), U8A_WRAP_PREFIX) && u8aEq(u8a.slice(-U8A_WRAP_POSTFIX.length), U8A_WRAP_POSTFIX) || withEthereum && u8a.length >= U8A_WRAP_ETHEREUM.length && u8aEq(u8a.subarray(0, U8A_WRAP_ETHEREUM.length), U8A_WRAP_ETHEREUM);
  }
  function u8aUnwrapBytes(bytes) {
    const u8a = u8aToU8a(bytes);
    return u8aIsWrapped(u8a, false) ? u8a.subarray(U8A_WRAP_PREFIX.length, u8a.length - U8A_WRAP_POSTFIX.length) : u8a;
  }
  function u8aWrapBytes(bytes) {
    const u8a = u8aToU8a(bytes);
    return u8aIsWrapped(u8a, true) ? u8a : u8aConcat(U8A_WRAP_PREFIX, u8a, U8A_WRAP_POSTFIX);
  }

  const DIV = BigInt(256);
  const NEG_MASK = BigInt(0xff);
  function createEmpty$1({
    bitLength = 0
  }) {
    return bitLength === -1 ? new Uint8Array() : new Uint8Array(Math.ceil(bitLength / 8));
  }
  function toU8a(value, {
    isLe,
    isNegative
  }) {
    const arr = [];
    if (isNegative) {
      value = (value + _1n) * -_1n;
    }
    while (value !== _0n) {
      const mod = value % DIV;
      const val = Number(isNegative ? mod ^ NEG_MASK : mod);
      if (isLe) {
        arr.push(val);
      } else {
        arr.unshift(val);
      }
      value = (value - mod) / DIV;
    }
    return Uint8Array.from(arr);
  }
  function nToU8a(value, options) {
    const opts = objectSpread({
      bitLength: -1,
      isLe: true,
      isNegative: false
    }, options);
    const valueBi = nToBigInt(value);
    if (valueBi === _0n) {
      return createEmpty$1(opts);
    }
    const u8a = toU8a(valueBi, opts);
    if (opts.bitLength === -1) {
      return u8a;
    }
    const byteLength = Math.ceil((opts.bitLength || 0) / 8);
    const output = new Uint8Array(byteLength);
    if (opts.isNegative) {
      output.fill(0xff);
    }
    output.set(u8a, opts.isLe ? 0 : byteLength - u8a.length);
    return output;
  }

  const ZERO_STR$1 = '0x00';
  function nToHex(value, options) {
    return !value ? ZERO_STR$1 : u8aToHex(nToU8a(value, objectSpread(
    {
      isLe: false,
      isNegative: false
    }, options)));
  }

  const BN_ZERO = new BN(0);
  const BN_ONE = new BN(1);
  const BN_TWO = new BN(2);
  const BN_THREE = new BN(3);
  const BN_FOUR = new BN(4);
  const BN_FIVE = new BN(5);
  const BN_SIX = new BN(6);
  const BN_SEVEN = new BN(7);
  const BN_EIGHT = new BN(8);
  const BN_NINE = new BN(9);
  const BN_TEN = new BN(10);
  const BN_HUNDRED = new BN(100);
  const BN_THOUSAND = new BN(1000);
  const BN_MILLION = new BN(1000000);
  const BN_BILLION = new BN(1000000000);
  const BN_QUINTILL = BN_BILLION.mul(BN_BILLION);
  const BN_MAX_INTEGER = new BN(Number.MAX_SAFE_INTEGER);

  function find(items, cmp) {
    assert(items.length >= 1, 'Must provide one or more BN arguments');
    let result = items[0];
    for (let i = 1; i < items.length; i++) {
      result = cmp(result, items[i]);
    }
    return result;
  }
  function bnMax(...items) {
    return find(items, BN.max);
  }
  function bnMin(...items) {
    return find(items, BN.min);
  }

  function isBigInt(value) {
    return typeof value === 'bigint';
  }

  function bnToBn(value) {
    return BN.isBN(value) ? value : !value ? new BN(0) : isHex(value) ? hexToBn(value.toString()) : isBigInt(value) ? new BN(value.toString()) : isToBn(value) ? value.toBn() : isToBigInt(value) ? new BN(value.toBigInt().toString()) : new BN(value);
  }

  const SQRT_MAX_SAFE_INTEGER = new BN(94906265);
  function bnSqrt(value) {
    const n = bnToBn(value);
    assert(n.gte(BN_ZERO), 'square root of negative numbers is not supported');
    if (n.lte(BN_MAX_INTEGER)) {
      return new BN(Math.floor(Math.sqrt(n.toNumber())));
    }
    let x0 = SQRT_MAX_SAFE_INTEGER.clone();
    while (true) {
      const x1 = n.div(x0).iadd(x0).ishrn(1);
      if (x0.eq(x1) || x0.eq(x1.sub(BN_ONE))) {
        return x0;
      }
      x0 = x1;
    }
  }

  function isNumber(value) {
    return typeof value === 'number';
  }

  const DEFAULT_OPTS$1 = {
    bitLength: -1,
    isLe: true,
    isNegative: false
  };
  function createEmpty(byteLength, options) {
    return options.bitLength === -1 ? new Uint8Array() : new Uint8Array(byteLength);
  }
  function createValue(valueBn, byteLength, {
    isLe,
    isNegative
  }) {
    const output = new Uint8Array(byteLength);
    const bn = isNegative ? valueBn.toTwos(byteLength * 8) : valueBn;
    output.set(bn.toArray(isLe ? 'le' : 'be', byteLength), 0);
    return output;
  }
  function bnToU8a(value, arg1 = DEFAULT_OPTS$1, arg2) {
    const options = objectSpread({
      bitLength: -1,
      isLe: true,
      isNegative: false
    }, isNumber(arg1) ? {
      bitLength: arg1,
      isLe: arg2
    } : arg1);
    const valueBn = bnToBn(value);
    const byteLength = options.bitLength === -1 ? Math.ceil(valueBn.bitLength() / 8) : Math.ceil((options.bitLength || 0) / 8);
    return value ? createValue(valueBn, byteLength, options) : createEmpty(byteLength, options);
  }

  const ZERO_STR = '0x00';
  const DEFAULT_OPTS = {
    bitLength: -1,
    isLe: false,
    isNegative: false
  };
  function bnToHex(value, arg1 = DEFAULT_OPTS, arg2) {
    return !value ? ZERO_STR : u8aToHex(bnToU8a(value, objectSpread(
    {
      isLe: false,
      isNegative: false
    }, isNumber(arg1) ? {
      bitLength: arg1,
      isLe: arg2
    } : arg1)));
  }

  function bufferToU8a(buffer) {
    return new Uint8Array(buffer || []);
  }

  const MAX_U8 = BN_TWO.pow(new BN(8 - 2)).isub(BN_ONE);
  const MAX_U16 = BN_TWO.pow(new BN(16 - 2)).isub(BN_ONE);
  const MAX_U32 = BN_TWO.pow(new BN(32 - 2)).isub(BN_ONE);
  function compactToU8a(value) {
    const bn = bnToBn(value);
    if (bn.lte(MAX_U8)) {
      return new Uint8Array([bn.toNumber() << 2]);
    } else if (bn.lte(MAX_U16)) {
      return bnToU8a(bn.shln(2).iadd(BN_ONE), 16, true);
    } else if (bn.lte(MAX_U32)) {
      return bnToU8a(bn.shln(2).iadd(BN_TWO), 32, true);
    }
    const u8a = bnToU8a(bn);
    let length = u8a.length;
    while (u8a[length - 1] === 0) {
      length--;
    }
    assert(length >= 4, 'Invalid length, previous checks match anything less than 2^30');
    return u8aConcat(
    [(length - 4 << 2) + 0b11], u8a.subarray(0, length));
  }

  function compactAddLength(input) {
    return u8aConcat(compactToU8a(input.length), input);
  }

  function compactFromU8a(input) {
    const u8a = u8aToU8a(input);
    const flag = u8a[0] & 0b11;
    if (flag === 0b00) {
      return [1, new BN(u8a[0]).ishrn(2)];
    } else if (flag === 0b01) {
      return [2, u8aToBn(u8a.subarray(0, 2), true).ishrn(2)];
    } else if (flag === 0b10) {
      return [4, u8aToBn(u8a.subarray(0, 4), true).ishrn(2)];
    }
    const offset = 1 + new BN(u8a[0]).ishrn(2)
    .iadd(BN_FOUR)
    .toNumber();
    return [offset, u8aToBn(u8a.subarray(1, offset), true)];
  }

  function compactStripLength(input) {
    const [offset, length] = compactFromU8a(input);
    const total = offset + length.toNumber();
    return [total, input.subarray(offset, total)];
  }

  const HRS = 60 * 60;
  const DAY = HRS * 24;
  const ZERO = {
    days: 0,
    hours: 0,
    milliseconds: 0,
    minutes: 0,
    seconds: 0
  };
  function addTime(a, b) {
    return {
      days: a.days + b.days,
      hours: a.hours + b.hours,
      milliseconds: a.milliseconds + b.milliseconds,
      minutes: a.minutes + b.minutes,
      seconds: a.seconds + b.seconds
    };
  }
  function extractDays(milliseconds, hrs) {
    const days = Math.floor(hrs / 24);
    return addTime(objectSpread({}, ZERO, {
      days
    }), extractTime(milliseconds - days * DAY * 1000));
  }
  function extractHrs(milliseconds, mins) {
    const hrs = mins / 60;
    if (hrs < 24) {
      const hours = Math.floor(hrs);
      return addTime(objectSpread({}, ZERO, {
        hours
      }), extractTime(milliseconds - hours * HRS * 1000));
    }
    return extractDays(milliseconds, hrs);
  }
  function extractMins(milliseconds, secs) {
    const mins = secs / 60;
    if (mins < 60) {
      const minutes = Math.floor(mins);
      return addTime(objectSpread({}, ZERO, {
        minutes
      }), extractTime(milliseconds - minutes * 60 * 1000));
    }
    return extractHrs(milliseconds, mins);
  }
  function extractSecs(milliseconds) {
    const secs = milliseconds / 1000;
    if (secs < 60) {
      const seconds = Math.floor(secs);
      return addTime(objectSpread({}, ZERO, {
        seconds
      }), extractTime(milliseconds - seconds * 1000));
    }
    return extractMins(milliseconds, secs);
  }
  function extractTime(milliseconds) {
    return !milliseconds ? ZERO : milliseconds < 1000 ? objectSpread({}, ZERO, {
      milliseconds
    }) : extractSecs(milliseconds);
  }

  const NUMBER_REGEX = new RegExp('(\\d+?)(?=(\\d{3})+(?!\\d)|$)', 'g');
  function formatDecimal(value) {
    const isNegative = value[0].startsWith('-');
    const matched = isNegative ? value.substr(1).match(NUMBER_REGEX) : value.match(NUMBER_REGEX);
    return matched ? `${isNegative ? '-' : ''}${matched.join(',')}` : value;
  }

  const SI_MID = 8;
  const SI = [{
    power: -24,
    text: 'yocto',
    value: 'y'
  }, {
    power: -21,
    text: 'zepto',
    value: 'z'
  }, {
    power: -18,
    text: 'atto',
    value: 'a'
  }, {
    power: -15,
    text: 'femto',
    value: 'f'
  }, {
    power: -12,
    text: 'pico',
    value: 'p'
  }, {
    power: -9,
    text: 'nano',
    value: 'n'
  }, {
    power: -6,
    text: 'micro',
    value: 'µ'
  }, {
    power: -3,
    text: 'milli',
    value: 'm'
  }, {
    power: 0,
    text: 'Unit',
    value: '-'
  },
  {
    power: 3,
    text: 'Kilo',
    value: 'k'
  }, {
    power: 6,
    text: 'Mill',
    value: 'M'
  },
  {
    power: 9,
    text: 'Bill',
    value: 'B'
  },
  {
    power: 12,
    text: 'Tril',
    value: 'T'
  },
  {
    power: 15,
    text: 'Peta',
    value: 'P'
  }, {
    power: 18,
    text: 'Exa',
    value: 'E'
  }, {
    power: 21,
    text: 'Zeta',
    value: 'Z'
  }, {
    power: 24,
    text: 'Yotta',
    value: 'Y'
  }];
  function findSi(type) {
    for (let i = 0; i < SI.length; i++) {
      if (SI[i].value === type) {
        return SI[i];
      }
    }
    return SI[SI_MID];
  }
  function calcSi(text, decimals, forceUnit) {
    if (forceUnit) {
      return findSi(forceUnit);
    }
    const siDefIndex = SI_MID - 1 + Math.ceil((text.length - decimals) / 3);
    return SI[siDefIndex] || SI[siDefIndex < 0 ? 0 : SI.length - 1];
  }

  const DEFAULT_DECIMALS = 0;
  const DEFAULT_UNIT = SI[SI_MID].text;
  let defaultDecimals = DEFAULT_DECIMALS;
  let defaultUnit = DEFAULT_UNIT;
  function getUnits(si, withSi, withSiFull, withUnit) {
    const unit = isBoolean(withUnit) ? SI[SI_MID].text : withUnit;
    return withSi || withSiFull ? si.value === '-' ? withUnit ? ` ${unit}` : '' : ` ${withSiFull ? `${si.text}${withUnit ? ' ' : ''}` : si.value}${withUnit ? unit : ''}` : '';
  }
  function getPrePost(text, decimals, forceUnit) {
    const si = calcSi(text, decimals, forceUnit);
    const mid = text.length - (decimals + si.power);
    const prefix = text.substring(0, mid);
    const padding = mid < 0 ? 0 - mid : 0;
    const postfix = `${`${new Array(padding + 1).join('0')}${text}`.substring(mid < 0 ? 0 : mid)}0000`.substring(0, 4);
    return [si, prefix || '0', postfix];
  }
  function _formatBalance(input, options = true, optDecimals = defaultDecimals) {
    let text = bnToBn(input).toString();
    if (text.length === 0 || text === '0') {
      return '0';
    }
    const {
      decimals = optDecimals,
      forceUnit = undefined,
      withSi = true,
      withSiFull = false,
      withUnit = true
    } = isBoolean(options) ? {
      withSi: options
    } : options;
    let sign = '';
    if (text[0].startsWith('-')) {
      sign = '-';
      text = text.substring(1);
    }
    const [si, prefix, postfix] = getPrePost(text, decimals, forceUnit);
    const units = getUnits(si, withSi, withSiFull, withUnit);
    return `${sign}${formatDecimal(prefix)}.${postfix}${units}`;
  }
  const formatBalance = _formatBalance;
  formatBalance.calcSi = (text, decimals = defaultDecimals) => calcSi(text, decimals);
  formatBalance.findSi = findSi;
  formatBalance.getDefaults = () => {
    return {
      decimals: defaultDecimals,
      unit: defaultUnit
    };
  };
  formatBalance.getOptions = (decimals = defaultDecimals) => {
    return SI.filter(({
      power
    }) => power < 0 ? decimals + power >= 0 : true);
  };
  formatBalance.setDefaults = ({
    decimals,
    unit
  }) => {
    defaultDecimals = isUndefined(decimals) ? defaultDecimals : Array.isArray(decimals) ? decimals[0] : decimals;
    defaultUnit = isUndefined(unit) ? defaultUnit : Array.isArray(unit) ? unit[0] : unit;
    SI[SI_MID].text = defaultUnit;
  };

  function zeroPad(value) {
    return value.toString().padStart(2, '0');
  }
  function formatDate(date) {
    const year = date.getFullYear().toString();
    const month = zeroPad(date.getMonth() + 1);
    const day = zeroPad(date.getDate());
    const hour = zeroPad(date.getHours());
    const minute = zeroPad(date.getMinutes());
    const second = zeroPad(date.getSeconds());
    return `${year}-${month}-${day} ${hour}:${minute}:${second}`;
  }

  function formatValue(elapsed) {
    if (elapsed < 15) {
      return `${elapsed.toFixed(1)}s`;
    } else if (elapsed < 60) {
      return `${elapsed | 0}s`;
    } else if (elapsed < 3600) {
      return `${elapsed / 60 | 0}m`;
    }
    return `${elapsed / 3600 | 0}h`;
  }
  function formatElapsed(now, value) {
    const tsNow = now && now.getTime() || 0;
    const tsValue = value instanceof Date ? value.getTime() : bnToBn(value).toNumber();
    return tsNow && tsValue ? formatValue(Math.max(Math.abs(tsNow - tsValue), 0) / 1000) : '0.0s';
  }

  function formatNumber(value) {
    return formatDecimal(bnToBn(value).toString());
  }

  function hexHasPrefix(value) {
    return !!value && isHex(value, -1);
  }

  function hexAddPrefix(value) {
    return value && hexHasPrefix(value) ? value : `0x${value && value.length % 2 === 1 ? '0' : ''}${value || ''}`;
  }

  function hexFixLength(value, bitLength = -1, withPadding = false) {
    const strLength = Math.ceil(bitLength / 4);
    const hexLength = strLength + 2;
    return hexAddPrefix(bitLength === -1 || value.length === hexLength || !withPadding && value.length < hexLength ? hexStripPrefix(value) : value.length > hexLength ? hexStripPrefix(value).slice(-1 * strLength) : `${'0'.repeat(strLength)}${hexStripPrefix(value)}`.slice(-1 * strLength));
  }

  function hexToNumber(value) {
    return value ? hexToBn(value).toNumber() : NaN;
  }

  function hexToString(_value) {
    return u8aToString(hexToU8a(_value));
  }

  function isArray(value) {
    return Array.isArray(value);
  }

  function isString(value) {
    return typeof value === 'string' || value instanceof String;
  }

  const FORMAT = [9, 10, 13];
  function isAsciiByte(b) {
    return b < 127 && (b >= 32 || FORMAT.includes(b));
  }
  function isAsciiChar(s) {
    return isAsciiByte(s.charCodeAt(0));
  }
  function isAscii(value) {
    const isStringIn = isString(value);
    if (value) {
      return isStringIn && !isHex(value) ? value.toString().split('').every(isAsciiChar) : u8aToU8a(value).every(isAsciiByte);
    }
    return isStringIn;
  }

  function isChildClass(Parent, Child) {
    return Child
    ? Parent === Child || Parent.isPrototypeOf(Child) : false;
  }

  const checkCodec = isOnObject('toHex', 'toU8a');
  const checkRegistry = isOnObject('get');
  function isCodec(value) {
    return checkCodec(value) && checkRegistry(value.registry);
  }

  const checker = isOnObject('toBigInt', 'toBn', 'toNumber', 'unwrap');
  function isCompact(value) {
    return checker(value);
  }

  function isError(value) {
    return value instanceof Error;
  }

  function isInstanceOf(value, clazz) {
    return value instanceof clazz;
  }

  const word = '[a-fA-F\\d:]';
  const b = options => options && options.includeBoundaries ?
  	`(?:(?<=\\s|^)(?=${word})|(?<=${word})(?=\\s|$))` :
  	'';
  const v4 = '(?:25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]\\d|\\d)(?:\\.(?:25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]\\d|\\d)){3}';
  const v6seg = '[a-fA-F\\d]{1,4}';
  const v6 = `
(?:
(?:${v6seg}:){7}(?:${v6seg}|:)|                                    // 1:2:3:4:5:6:7::  1:2:3:4:5:6:7:8
(?:${v6seg}:){6}(?:${v4}|:${v6seg}|:)|                             // 1:2:3:4:5:6::    1:2:3:4:5:6::8   1:2:3:4:5:6::8  1:2:3:4:5:6::1.2.3.4
(?:${v6seg}:){5}(?::${v4}|(?::${v6seg}){1,2}|:)|                   // 1:2:3:4:5::      1:2:3:4:5::7:8   1:2:3:4:5::8    1:2:3:4:5::7:1.2.3.4
(?:${v6seg}:){4}(?:(?::${v6seg}){0,1}:${v4}|(?::${v6seg}){1,3}|:)| // 1:2:3:4::        1:2:3:4::6:7:8   1:2:3:4::8      1:2:3:4::6:7:1.2.3.4
(?:${v6seg}:){3}(?:(?::${v6seg}){0,2}:${v4}|(?::${v6seg}){1,4}|:)| // 1:2:3::          1:2:3::5:6:7:8   1:2:3::8        1:2:3::5:6:7:1.2.3.4
(?:${v6seg}:){2}(?:(?::${v6seg}){0,3}:${v4}|(?::${v6seg}){1,5}|:)| // 1:2::            1:2::4:5:6:7:8   1:2::8          1:2::4:5:6:7:1.2.3.4
(?:${v6seg}:){1}(?:(?::${v6seg}){0,4}:${v4}|(?::${v6seg}){1,6}|:)| // 1::              1::3:4:5:6:7:8   1::8            1::3:4:5:6:7:1.2.3.4
(?::(?:(?::${v6seg}){0,5}:${v4}|(?::${v6seg}){1,7}|:))             // ::2:3:4:5:6:7:8  ::2:3:4:5:6:7:8  ::8             ::1.2.3.4
)(?:%[0-9a-zA-Z]{1,})?                                             // %eth0            %1
`.replace(/\s*\/\/.*$/gm, '').replace(/\n/g, '').trim();
  const v46Exact = new RegExp(`(?:^${v4}$)|(?:^${v6}$)`);
  const v4exact = new RegExp(`^${v4}$`);
  const v6exact = new RegExp(`^${v6}$`);
  const ip = options => options && options.exact ?
  	v46Exact :
  	new RegExp(`(?:${b(options)}${v4}${b(options)})|(?:${b(options)}${v6}${b(options)})`, 'g');
  ip.v4 = options => options && options.exact ? v4exact : new RegExp(`${b(options)}${v4}${b(options)}`, 'g');
  ip.v6 = options => options && options.exact ? v6exact : new RegExp(`${b(options)}${v6}${b(options)}`, 'g');
  var ipRegex = ip;
  const ipRegex$1 = ipRegex;

  function isIp(value, type) {
    if (type === 'v4') {
      return ipRegex$1.v4({
        exact: true
      }).test(value);
    } else if (type === 'v6') {
      return ipRegex$1.v6({
        exact: true
      }).test(value);
    }
    return ipRegex$1({
      exact: true
    }).test(value);
  }

  function replacer(_, v) {
    return isBigInt(v) ? v.toString() : v;
  }
  function stringify(value, space) {
    return JSON.stringify(value, replacer, space);
  }

  function isJsonObject(value) {
    const str = typeof value !== 'string' ? stringify(value) : value;
    try {
      const obj = JSON.parse(str);
      return typeof obj === 'object' && obj !== null;
    } catch (e) {
      return false;
    }
  }

  const isObservable = isOn('next');

  const isPromise = isOnObject('catch', 'then');

  const REGEX_DEV = /(Development|Local Testnet)$/;
  function isTestChain(chain) {
    if (!chain) {
      return false;
    }
    return !!REGEX_DEV.test(chain.toString());
  }

  function isUtf8(value) {
    if (!value) {
      return isString(value);
    }
    const u8a = u8aToU8a(value);
    const len = u8a.length;
    let i = 0;
    while (i < len) {
      if (u8a[i] <= 0x7F)
        {
          i += 1;
        } else if (u8a[i] >= 0xC2 && u8a[i] <= 0xDF)
        {
          if (i + 1 < len)
            {
              if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0xBF) {
                return false;
              }
            } else {
            return false;
          }
          i += 2;
        } else if (u8a[i] === 0xE0)
        {
          if (i + 2 < len)
            {
              if (u8a[i + 1] < 0xA0 || u8a[i + 1] > 0xBF) {
                return false;
              }
              if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
                return false;
              }
            } else {
            return false;
          }
          i += 3;
        } else if (u8a[i] >= 0xE1 && u8a[i] <= 0xEC)
        {
          if (i + 2 < len)
            {
              if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0xBF) {
                return false;
              }
              if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
                return false;
              }
            } else {
            return false;
          }
          i += 3;
        } else if (u8a[i] === 0xED)
        {
          if (i + 2 < len)
            {
              if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0x9F) {
                return false;
              }
              if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
                return false;
              }
            } else {
            return false;
          }
          i += 3;
        } else if (u8a[i] >= 0xEE && u8a[i] <= 0xEF)
        {
          if (i + 2 < len)
            {
              if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0xBF) {
                return false;
              }
              if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
                return false;
              }
            } else {
            return false;
          }
          i += 3;
        } else if (u8a[i] === 0xF0)
        {
          if (i + 3 < len)
            {
              if (u8a[i + 1] < 0x90 || u8a[i + 1] > 0xBF) {
                return false;
              }
              if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
                return false;
              }
              if (u8a[i + 3] < 0x80 || u8a[i + 3] > 0xBF) {
                return false;
              }
            } else {
            return false;
          }
          i += 4;
        } else if (u8a[i] >= 0xF1 && u8a[i] <= 0xF3)
        {
          if (i + 3 < len)
            {
              if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0xBF) {
                return false;
              }
              if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
                return false;
              }
              if (u8a[i + 3] < 0x80 || u8a[i + 3] > 0xBF) {
                return false;
              }
            } else {
            return false;
          }
          i += 4;
        } else if (u8a[i] === 0xF4)
        {
          if (i + 3 < len)
            {
              if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0x8F) {
                return false;
              }
              if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
                return false;
              }
              if (u8a[i + 3] < 0x80 || u8a[i + 3] > 0xBF) {
                return false;
              }
            } else {
            return false;
          }
          i += 4;
        } else {
        return false;
      }
    }
    return true;
  }

  const WASM_MAGIC = new Uint8Array([0, 97, 115, 109]);
  function isWasm(value) {
    return isU8a(value) && u8aEq(value.subarray(0, 4), WASM_MAGIC);
  }

  function lazyMethod(result, item, creator, getName) {
    const name = getName ? getName(item) : item.toString();
    let value;
    Object.defineProperty(result, name, {
      configurable: true,
      enumerable: true,
      get: function () {
        if (isUndefined(value)) {
          value = creator(item);
          try {
            Object.defineProperty(this, name, {
              value
            });
          } catch {
          }
        }
        return value;
      }
    });
  }
  function lazyMethods(result, items, creator, getName) {
    for (let i = 0; i < items.length; i++) {
      lazyMethod(result, items[i], creator, getName);
    }
    return result;
  }

  const logTo = {
    debug: 'log',
    error: 'error',
    log: 'log',
    warn: 'warn'
  };
  function formatOther(value) {
    if (value && isObject(value) && value.constructor === Object) {
      const result = {};
      for (const k of Object.keys(value)) {
        result[k] = loggerFormat(value[k]);
      }
      return result;
    }
    return value;
  }
  function loggerFormat(value) {
    if (Array.isArray(value)) {
      return value.map(loggerFormat);
    } else if (isBn(value)) {
      return value.toString();
    } else if (isU8a(value) || isBuffer(value)) {
      return u8aToHex(u8aToU8a(value));
    }
    return formatOther(value);
  }
  function formatWithLength(maxLength) {
    return v => {
      if (maxLength <= 0) {
        return v;
      }
      const r = `${v}`;
      return r.length < maxLength ? v : `${r.substr(0, maxLength)} ...`;
    };
  }
  function apply(log, type, values, maxSize = -1) {
    if (values.length === 1 && isFunction(values[0])) {
      const fnResult = values[0]();
      return apply(log, type, Array.isArray(fnResult) ? fnResult : [fnResult], maxSize);
    }
    console[logTo[log]](formatDate(new Date()), type, ...values.map(loggerFormat).map(formatWithLength(maxSize)));
  }
  function noop() {
  }
  function isDebugOn(e, type) {
    return !!e && (e === '*' || type === e || e.endsWith('*') && type.startsWith(e.slice(0, -1)));
  }
  function isDebugOff(e, type) {
    return !!e && e.startsWith('-') && (type === e.slice(1) || e.endsWith('*') && type.startsWith(e.slice(1, -1)));
  }
  function getDebugFlag(env, type) {
    let flag = false;
    for (const e of env) {
      if (isDebugOn(e, type)) {
        flag = true;
      } else if (isDebugOff(e, type)) {
        flag = false;
      }
    }
    return flag;
  }
  function parseEnv(type) {
    const env = (hasProcess ? process : {}).env || {};
    const maxSize = parseInt(env.DEBUG_MAX || '-1', 10);
    return [getDebugFlag((env.DEBUG || '').toLowerCase().split(','), type), isNaN(maxSize) ? -1 : maxSize];
  }
  function logger(_type) {
    const type = `${_type.toUpperCase()}:`.padStart(16);
    const [isDebug, maxSize] = parseEnv(_type.toLowerCase());
    return {
      debug: isDebug ? (...values) => apply('debug', type, values, maxSize) : noop,
      error: (...values) => apply('error', type, values),
      log: (...values) => apply('log', type, values),
      noop,
      warn: (...values) => apply('warn', type, values)
    };
  }

  function defaultGetId() {
    return 'none';
  }
  function memoize(fn, {
    getInstanceId = defaultGetId
  } = {}) {
    const cache = {};
    const memoized = (...args) => {
      const stringParams = stringify(args);
      const instanceId = getInstanceId();
      if (!cache[instanceId]) {
        cache[instanceId] = {};
      }
      if (isUndefined(cache[instanceId][stringParams])) {
        cache[instanceId][stringParams] = fn(...args);
      }
      return cache[instanceId][stringParams];
    };
    memoized.unmemoize = (...args) => {
      const stringParams = stringify(args);
      const instanceId = getInstanceId();
      if (cache[instanceId] && !isUndefined(cache[instanceId][stringParams])) {
        delete cache[instanceId][stringParams];
      }
    };
    return memoized;
  }

  function numberToHex(value, bitLength = -1) {
    if (isUndefined(value) || isNull(value) || isNaN(value)) {
      return '0x';
    }
    const hex = value.toString(16);
    return hexFixLength(hex.length % 2 ? `0${hex}` : hex, bitLength, true);
  }

  function numberToU8a(value, bitLength = -1) {
    return isUndefined(value) || isNull(value) || isNaN(value) ? new Uint8Array() : hexToU8a(numberToHex(value, bitLength));
  }

  function objectClear(value) {
    const keys = Object.keys(value);
    for (let i = 0; i < keys.length; i++) {
      delete value[keys[i]];
    }
    return value;
  }

  function objectCopy(source) {
    return objectSpread({}, source);
  }

  function objectEntries(obj) {
    return Object.entries(obj);
  }

  function objectProperty(that, key, getter) {
    if (!Object.prototype.hasOwnProperty.call(that, key) && isUndefined(that[key])) {
      Object.defineProperty(that, key, {
        enumerable: true,
        get: () => getter(key)
      });
    }
  }
  function objectProperties(that, keys, getter) {
    for (let i = 0; i < keys.length; i++) {
      objectProperty(that, keys[i], k => getter(k, i));
    }
  }

  function objectValues(obj) {
    return Object.values(obj);
  }

  function promisify(self, fn, ...params) {
    return new Promise((resolve, reject) => {
      fn.apply(self, params.concat((error, result) => {
        if (error) {
          reject(error);
        } else {
          resolve(result);
        }
      }));
    });
  }

  function converter$1(fn) {
    const format = (w, i) => fn(w[0], i) + w.slice(1);
    return value => value.toString()
    .replace(/[-_., ]+/g, ' ')
    .trim()
    .split(' ')
    .map((w, i) => format(w.toUpperCase() === w
    ? w.toLowerCase()
    : w.replace(/^[A-Z0-9]{2,}[^a-z]/, w => w.slice(0, w.length - 1).toLowerCase() + w.slice(-1).toUpperCase()), i))
    .join('');
  }
  const stringCamelCase = converter$1((w, i) => i ? w.toUpperCase() : w.toLowerCase());
  const stringPascalCase = converter$1(w => w.toUpperCase());

  function converter(fn) {
    return value => value ? fn(value[0]) + value.slice(1) : '';
  }
  const stringLowerFirst = converter(s => s.toLowerCase());
  const stringUpperFirst = converter(s => s.toUpperCase());

  function stringShorten(value, prefixLength = 6) {
    return value.length <= 2 + 2 * prefixLength ? value.toString() : `${value.substr(0, prefixLength)}…${value.slice(-prefixLength)}`;
  }

  function stringToHex(value) {
    return u8aToHex(stringToU8a(value));
  }

  const DEDUPE = 'Either remove and explicitly install matching versions or dedupe using your package manager.\nThe following conflicting packages were found:';
  function getEntry(name) {
    const _global = xglobal;
    if (!_global.__polkadotjs) {
      _global.__polkadotjs = {};
    }
    if (!_global.__polkadotjs[name]) {
      _global.__polkadotjs[name] = [];
    }
    return _global.__polkadotjs[name];
  }
  function getVersionLength(all) {
    let length = 0;
    for (const {
      version
    } of all) {
      length = Math.max(length, version.length);
    }
    return length;
  }
  function flattenInfos(all) {
    const verLength = getVersionLength(all);
    const stringify = ({
      name,
      version
    }) => `\t${version.padEnd(verLength)}\t${name}`;
    return all.map(stringify).join('\n');
  }
  function flattenVersions(entry) {
    const toPath = version => isString(version) ? {
      version
    } : version;
    const all = entry.map(toPath);
    const verLength = getVersionLength(all);
    const stringify = ({
      path,
      type,
      version
    }) => `\t${`${type || ''}`.padStart(3)} ${version.padEnd(verLength)}\t${!path || path.length < 5 ? '<unknown>' : path}`;
    return all.map(stringify).join('\n');
  }
  function getPath(infoPath, pathOrFn) {
    if (infoPath) {
      return infoPath;
    } else if (isFunction(pathOrFn)) {
      try {
        return pathOrFn() || '';
      } catch (error) {
        return '';
      }
    }
    return pathOrFn || '';
  }
  function detectPackage({
    name,
    path,
    type,
    version
  }, pathOrFn, deps = []) {
    assert(name.startsWith('@polkadot'), () => `Invalid package descriptor ${name}`);
    const entry = getEntry(name);
    entry.push({
      path: getPath(path, pathOrFn),
      type,
      version
    });
    if (entry.length !== 1) {
      console.warn(`${name} has multiple versions, ensure that there is only one installed.\n${DEDUPE}\n${flattenVersions(entry)}`);
    } else {
      const mismatches = deps.filter(d => d && d.version !== version);
      if (mismatches.length) {
        console.warn(`${name} requires direct dependencies exactly matching version ${version}.\n${DEDUPE}\n${flattenInfos(mismatches)}`);
      }
    }
  }

  exports.BN = BN;
  exports.BN_BILLION = BN_BILLION;
  exports.BN_EIGHT = BN_EIGHT;
  exports.BN_FIVE = BN_FIVE;
  exports.BN_FOUR = BN_FOUR;
  exports.BN_HUNDRED = BN_HUNDRED;
  exports.BN_MAX_INTEGER = BN_MAX_INTEGER;
  exports.BN_MILLION = BN_MILLION;
  exports.BN_NINE = BN_NINE;
  exports.BN_ONE = BN_ONE;
  exports.BN_QUINTILL = BN_QUINTILL;
  exports.BN_SEVEN = BN_SEVEN;
  exports.BN_SIX = BN_SIX;
  exports.BN_TEN = BN_TEN;
  exports.BN_THOUSAND = BN_THOUSAND;
  exports.BN_THREE = BN_THREE;
  exports.BN_TWO = BN_TWO;
  exports.BN_ZERO = BN_ZERO;
  exports.U8A_WRAP_ETHEREUM = U8A_WRAP_ETHEREUM;
  exports.U8A_WRAP_POSTFIX = U8A_WRAP_POSTFIX;
  exports.U8A_WRAP_PREFIX = U8A_WRAP_PREFIX;
  exports._0n = _0n;
  exports._1Bn = _1Bn;
  exports._1Mn = _1Mn;
  exports._1Qn = _1Qn;
  exports._1n = _1n;
  exports._2pow53n = _2pow53n;
  exports.arrayChunk = arrayChunk;
  exports.arrayFilter = arrayFilter;
  exports.arrayFlatten = arrayFlatten;
  exports.arrayRange = arrayRange;
  exports.arrayShuffle = arrayShuffle;
  exports.arrayZip = arrayZip;
  exports.assert = assert;
  exports.assertReturn = assertReturn;
  exports.assertUnreachable = assertUnreachable;
  exports.bnFromHex = hexToBn;
  exports.bnMax = bnMax;
  exports.bnMin = bnMin;
  exports.bnSqrt = bnSqrt;
  exports.bnToBn = bnToBn;
  exports.bnToHex = bnToHex;
  exports.bnToU8a = bnToU8a;
  exports.bufferToU8a = bufferToU8a;
  exports.calcSi = calcSi;
  exports.compactAddLength = compactAddLength;
  exports.compactFromU8a = compactFromU8a;
  exports.compactStripLength = compactStripLength;
  exports.compactToU8a = compactToU8a;
  exports.detectPackage = detectPackage;
  exports.extractTime = extractTime;
  exports.findSi = findSi;
  exports.formatBalance = formatBalance;
  exports.formatDate = formatDate;
  exports.formatDecimal = formatDecimal;
  exports.formatElapsed = formatElapsed;
  exports.formatNumber = formatNumber;
  exports.hasBigInt = hasBigInt;
  exports.hasBuffer = hasBuffer;
  exports.hasCjs = hasCjs;
  exports.hasDirname = hasDirname;
  exports.hasEsm = hasEsm;
  exports.hasProcess = hasProcess;
  exports.hasWasm = hasWasm;
  exports.hexAddPrefix = hexAddPrefix;
  exports.hexFixLength = hexFixLength;
  exports.hexHasPrefix = hexHasPrefix;
  exports.hexStripPrefix = hexStripPrefix;
  exports.hexToBigInt = hexToBigInt;
  exports.hexToBn = hexToBn;
  exports.hexToNumber = hexToNumber;
  exports.hexToString = hexToString;
  exports.hexToU8a = hexToU8a;
  exports.isArray = isArray;
  exports.isAscii = isAscii;
  exports.isBigInt = isBigInt;
  exports.isBn = isBn;
  exports.isBoolean = isBoolean;
  exports.isBuffer = isBuffer;
  exports.isChildClass = isChildClass;
  exports.isCodec = isCodec;
  exports.isCompact = isCompact;
  exports.isError = isError;
  exports.isFunction = isFunction;
  exports.isHex = isHex;
  exports.isInstanceOf = isInstanceOf;
  exports.isIp = isIp;
  exports.isJsonObject = isJsonObject;
  exports.isNull = isNull;
  exports.isNumber = isNumber;
  exports.isObject = isObject;
  exports.isObservable = isObservable;
  exports.isPromise = isPromise;
  exports.isString = isString;
  exports.isTestChain = isTestChain;
  exports.isToBigInt = isToBigInt;
  exports.isToBn = isToBn;
  exports.isU8a = isU8a;
  exports.isUndefined = isUndefined;
  exports.isUtf8 = isUtf8;
  exports.isWasm = isWasm;
  exports.lazyMethod = lazyMethod;
  exports.lazyMethods = lazyMethods;
  exports.logger = logger;
  exports.loggerFormat = loggerFormat;
  exports.memoize = memoize;
  exports.nMax = nMax;
  exports.nMin = nMin;
  exports.nSqrt = nSqrt;
  exports.nToBigInt = nToBigInt;
  exports.nToHex = nToHex;
  exports.nToU8a = nToU8a;
  exports.numberToHex = numberToHex;
  exports.numberToU8a = numberToU8a;
  exports.objectClear = objectClear;
  exports.objectCopy = objectCopy;
  exports.objectEntries = objectEntries;
  exports.objectKeys = objectKeys;
  exports.objectProperties = objectProperties;
  exports.objectProperty = objectProperty;
  exports.objectSpread = objectSpread;
  exports.objectValues = objectValues;
  exports.packageInfo = packageInfo;
  exports.promisify = promisify;
  exports.stringCamelCase = stringCamelCase;
  exports.stringLowerFirst = stringLowerFirst;
  exports.stringPascalCase = stringPascalCase;
  exports.stringShorten = stringShorten;
  exports.stringToHex = stringToHex;
  exports.stringToU8a = stringToU8a;
  exports.stringUpperFirst = stringUpperFirst;
  exports.stringify = stringify;
  exports.u8aCmp = u8aCmp;
  exports.u8aConcat = u8aConcat;
  exports.u8aEmpty = u8aEmpty;
  exports.u8aEq = u8aEq;
  exports.u8aFixLength = u8aFixLength;
  exports.u8aIsWrapped = u8aIsWrapped;
  exports.u8aSorted = u8aSorted;
  exports.u8aToBigInt = u8aToBigInt;
  exports.u8aToBn = u8aToBn;
  exports.u8aToBuffer = u8aToBuffer;
  exports.u8aToHex = u8aToHex;
  exports.u8aToString = u8aToString;
  exports.u8aToU8a = u8aToU8a;
  exports.u8aUnwrapBytes = u8aUnwrapBytes;
  exports.u8aWrapBytes = u8aWrapBytes;

  Object.defineProperty(exports, '__esModule', { value: true });

}));
