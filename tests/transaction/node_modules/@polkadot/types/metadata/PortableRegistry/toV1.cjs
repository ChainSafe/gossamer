"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.toV1 = toV1;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
function convertType(key) {
  return (registry, _ref) => {
    let {
      type
    } = _ref;
    return registry.createType('Si1TypeDef', {
      [key]: {
        type: type.toNumber()
      }
    });
  };
}

function convertArray(registry, _ref2) {
  let {
    len,
    type
  } = _ref2;
  return registry.createType('Si1TypeDef', {
    Array: {
      len,
      type: type.toNumber()
    }
  });
}

function convertBitSequence(registry, _ref3) {
  let {
    bitOrderType,
    bitStoreType
  } = _ref3;
  return registry.createType('Si1TypeDef', {
    BitSequence: {
      bitOrderType: bitOrderType.toNumber(),
      bitStoreType: bitStoreType.toNumber()
    }
  });
}

const convertCompact = convertType('Compact');

function convertComposite(registry, _ref4) {
  let {
    fields
  } = _ref4;
  return registry.createType('Si1TypeDef', {
    Composite: {
      fields: convertFields(registry, fields)
    }
  });
}

function convertFields(registry, fields) {
  return fields.map(_ref5 => {
    let {
      docs,
      name,
      type,
      typeName
    } = _ref5;
    return registry.createType('Si1Field', {
      docs,
      name,
      type: type.toNumber(),
      typeName
    });
  });
}

function convertPhantom(registry, path) {
  console.warn(`Converting phantom type ${path.map(p => p.toString()).join('::')} to empty tuple`);
  return registry.createType('Si1TypeDef', {
    Tuple: []
  });
}

function convertPrimitive(registry, prim) {
  return registry.createType('Si1TypeDef', {
    Primitive: prim.toString()
  });
}

const convertSequence = convertType('Sequence');

function convertTuple(registry, types) {
  return registry.createType('Si1TypeDef', {
    Tuple: types.map(t => t.toNumber())
  });
}

function convertVariant(registry, _ref6) {
  let {
    variants
  } = _ref6;
  return registry.createType('Si1TypeDef', {
    Variant: {
      variants: variants.map((_ref7, index) => {
        let {
          discriminant,
          docs,
          fields,
          name
        } = _ref7;
        return registry.createType('Si1Variant', {
          docs,
          fields: convertFields(registry, fields),
          index: discriminant.isSome ? discriminant.unwrap().toNumber() : index,
          name
        });
      })
    }
  });
}

function convertDef(registry, _ref8) {
  let {
    def,
    path
  } = _ref8;
  let result;

  switch (def.type) {
    case 'Array':
      result = convertArray(registry, def.asArray);
      break;

    case 'BitSequence':
      result = convertBitSequence(registry, def.asBitSequence);
      break;

    case 'Compact':
      result = convertCompact(registry, def.asCompact);
      break;

    case 'Composite':
      result = convertComposite(registry, def.asComposite);
      break;

    case 'Phantom':
      result = convertPhantom(registry, path);
      break;

    case 'Primitive':
      result = convertPrimitive(registry, def.asPrimitive);
      break;

    case 'Sequence':
      result = convertSequence(registry, def.asSequence);
      break;

    case 'Tuple':
      result = convertTuple(registry, def.asTuple);
      break;

    case 'Variant':
      result = convertVariant(registry, def.asVariant);
      break;

    default:
      (0, _util.assertUnreachable)(def.type);
  }

  return result;
}

function toV1(registry, types) {
  return types.map((t, index) => registry.createType('PortableType', {
    // offsets are +1 from v0
    id: index + 1,
    type: {
      def: convertDef(registry, t),
      docs: [],
      params: t.params.map(p => registry.createType('Si1TypeParameter', {
        type: p.toNumber()
      })),
      path: t.path.map(p => p.toString())
    }
  }));
}