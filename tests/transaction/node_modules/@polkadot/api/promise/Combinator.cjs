"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.Combinator = void 0;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0
class Combinator {
  #allHasFired = false;
  #callback;
  #fired = [];
  #fns = [];
  #isActive = true;
  #results = [];
  #subscriptions = [];

  constructor(fns, callback) {
    this.#callback = callback; // eslint-disable-next-line @typescript-eslint/require-await

    this.#subscriptions = fns.map(async (input, index) => {
      const [fn, ...args] = Array.isArray(input) ? input : [input];
      this.#fired.push(false);
      this.#fns.push(fn); // Not quite 100% how to have a variable number at the front here
      // eslint-disable-next-line @typescript-eslint/no-unsafe-return,@typescript-eslint/ban-types

      return fn(...args, this._createCallback(index));
    });
  }

  _allHasFired() {
    this.#allHasFired || (this.#allHasFired = this.#fired.filter(hasFired => !hasFired).length === 0);
    return this.#allHasFired;
  }

  _createCallback(index) {
    return value => {
      this.#fired[index] = true;
      this.#results[index] = value;

      this._triggerUpdate();
    };
  }

  _triggerUpdate() {
    if (!this.#isActive || !(0, _util.isFunction)(this.#callback) || !this._allHasFired()) {
      return;
    }

    try {
      // eslint-disable-next-line @typescript-eslint/no-floating-promises
      this.#callback(this.#results);
    } catch (error) {// swallow, we don't want the handler to trip us up
    }
  }

  unsubscribe() {
    if (!this.#isActive) {
      return;
    }

    this.#isActive = false; // eslint-disable-next-line @typescript-eslint/no-misused-promises

    this.#subscriptions.forEach(async subscription => {
      try {
        const unsubscribe = await subscription;

        if ((0, _util.isFunction)(unsubscribe)) {
          unsubscribe();
        }
      } catch (error) {// ignore
      }
    });
  }

}

exports.Combinator = Combinator;