// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { lazyMethod, objectSpread, stringCamelCase } from '@polkadot/util';
import { lazyVariants } from "../../../create/lazy.js";
import { getSiName } from "../../util/index.js";
import { objectNameToCamel } from "../util.js";
import { createUnchecked } from "./createUnchecked.js";
export function filterCallsSome({
  calls
}) {
  return calls.isSome;
}
export function createCallFunction(registry, lookup, variant, sectionName, sectionIndex) {
  const {
    fields,
    index
  } = variant;
  const args = new Array(fields.length);

  for (let a = 0; a < fields.length; a++) {
    const {
      name,
      type,
      typeName
    } = fields[a];
    args[a] = objectSpread({
      name: stringCamelCase(name.unwrapOr(`param${a}`)),
      type: getSiName(lookup, type)
    }, typeName.isSome ? {
      typeName: typeName.unwrap()
    } : null);
  }

  return createUnchecked(registry, sectionName, new Uint8Array([sectionIndex, index.toNumber()]), registry.createTypeUnsafe('FunctionMetadataLatest', [objectSpread({
    args
  }, variant)]));
}
/** @internal */

export function decorateExtrinsics(registry, {
  lookup,
  pallets
}, version) {
  const result = {};
  const filtered = pallets.filter(filterCallsSome);

  for (let i = 0; i < filtered.length; i++) {
    const {
      calls,
      index,
      name
    } = filtered[i];
    const sectionName = stringCamelCase(name);
    const sectionIndex = version >= 12 ? index.toNumber() : i;
    lazyMethod(result, sectionName, () => lazyVariants(lookup, calls.unwrap(), objectNameToCamel, variant => createCallFunction(registry, lookup, variant, sectionName, sectionIndex)));
  }

  return result;
}