// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
// eslint-disable-next-line prefer-regex-literals
const NUMBER_REGEX = new RegExp('(\\d+?)(?=(\\d{3})+(?!\\d)|$)', 'g');
export function formatDecimal(value) {
  // We can do this by adjusting the regx, however for the sake of clarity
  // we rather strip and re-add the negative sign in the output
  const isNegative = value[0].startsWith('-');
  const matched = isNegative ? value.substr(1).match(NUMBER_REGEX) : value.match(NUMBER_REGEX);
  return matched ? `${isNegative ? '-' : ''}${matched.join(',')}` : value;
}