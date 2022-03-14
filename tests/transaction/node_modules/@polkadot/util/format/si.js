// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
export const SI_MID = 8;
export const SI = [{
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
}, // position 8
{
  power: 3,
  text: 'Kilo',
  value: 'k'
}, {
  power: 6,
  text: 'Mill',
  value: 'M'
}, // Mega, M
{
  power: 9,
  text: 'Bill',
  value: 'B'
}, // Giga, G
{
  power: 12,
  text: 'Tril',
  value: 'T'
}, // Tera, T
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
}]; // Given a SI type (e.g. k, m, Y) find the SI definition

export function findSi(type) {
  // use a loop here, better RN support (which doesn't have [].find)
  for (let i = 0; i < SI.length; i++) {
    if (SI[i].value === type) {
      return SI[i];
    }
  }

  return SI[SI_MID];
}
export function calcSi(text, decimals, forceUnit) {
  if (forceUnit) {
    return findSi(forceUnit);
  }

  const siDefIndex = SI_MID - 1 + Math.ceil((text.length - decimals) / 3);
  return SI[siDefIndex] || SI[siDefIndex < 0 ? 0 : SI.length - 1];
}