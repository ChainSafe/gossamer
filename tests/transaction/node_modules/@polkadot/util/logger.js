// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { formatDate } from "./format/formatDate.js";
import { isBn } from "./is/bn.js";
import { isBuffer } from "./is/buffer.js";
import { isFunction } from "./is/function.js";
import { isObject } from "./is/object.js";
import { isU8a } from "./is/u8a.js";
import { u8aToHex } from "./u8a/toHex.js";
import { u8aToU8a } from "./u8a/toU8a.js";
import { hasProcess } from "./has.js";
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

export function loggerFormat(value) {
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

function noop() {// noop
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
/**
 * @name Logger
 * @summary Creates a consistent log interface for messages
 * @description
 * Returns a `Logger` that has `.log`, `.error`, `.warn` and `.debug` (controlled with environment `DEBUG=typeA,typeB`) methods. Logging is done with a consistent prefix (type of logger, date) followed by the actual message using the underlying console.
 * @example
 * <BR>
 *
 * ```javascript
 * import { logger } from '@polkadot';
 *
 * const l = logger('test');
 * ```
 */


export function logger(_type) {
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