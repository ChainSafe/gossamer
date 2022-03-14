// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
// Adapted from https://github.com/JulienPalard/is_utf8/blob/master/is_utf8.c
import { u8aToU8a } from "../u8a/toU8a.js";
import { isString } from "./string.js";
/**
 * @name isUtf8
 * @summary Tests if the input is valid Utf8
 * @description
 * Checks to see if the input string or Uint8Array is valid Utf8
 */

export function isUtf8(value) {
  if (!value) {
    return isString(value);
  }

  const u8a = u8aToU8a(value);
  const len = u8a.length;
  let i = 0;

  while (i < len) {
    if (u8a[i] <= 0x7F)
      /* 00..7F */
      {
        i += 1;
      } else if (u8a[i] >= 0xC2 && u8a[i] <= 0xDF)
      /* C2..DF 80..BF */
      {
        if (i + 1 < len)
          /* Expect a 2nd byte */
          {
            if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0xBF) {
              // *message = "After a first byte between C2 and DF, expecting a 2nd byte between 80 and BF";
              // *faulty_bytes = 2;
              return false;
            }
          } else {
          // *message = "After a first byte between C2 and DF, expecting a 2nd byte.";
          // *faulty_bytes = 1;
          return false;
        }

        i += 2;
      } else if (u8a[i] === 0xE0)
      /* E0 A0..BF 80..BF */
      {
        if (i + 2 < len)
          /* Expect a 2nd and 3rd byte */
          {
            if (u8a[i + 1] < 0xA0 || u8a[i + 1] > 0xBF) {
              // *message = "After a first byte of E0, expecting a 2nd byte between A0 and BF.";
              // *faulty_bytes = 2;
              return false;
            }

            if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
              // *message = "After a first byte of E0, expecting a 3nd byte between 80 and BF.";
              // *faulty_bytes = 3;
              return false;
            }
          } else {
          // *message = "After a first byte of E0, expecting two following bytes.";
          // *faulty_bytes = 1;
          return false;
        }

        i += 3;
      } else if (u8a[i] >= 0xE1 && u8a[i] <= 0xEC)
      /* E1..EC 80..BF 80..BF */
      {
        if (i + 2 < len)
          /* Expect a 2nd and 3rd byte */
          {
            if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0xBF) {
              // *message = "After a first byte between E1 and EC, expecting the 2nd byte between 80 and BF.";
              // *faulty_bytes = 2;
              return false;
            }

            if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
              // *message = "After a first byte between E1 and EC, expecting the 3rd byte between 80 and BF.";
              // *faulty_bytes = 3;
              return false;
            }
          } else {
          // *message = "After a first byte between E1 and EC, expecting two following bytes.";
          // *faulty_bytes = 1;
          return false;
        }

        i += 3;
      } else if (u8a[i] === 0xED)
      /* ED 80..9F 80..BF */
      {
        if (i + 2 < len)
          /* Expect a 2nd and 3rd byte */
          {
            if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0x9F) {
              // *message = "After a first byte of ED, expecting 2nd byte between 80 and 9F.";
              // *faulty_bytes = 2;
              return false;
            }

            if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
              // *message = "After a first byte of ED, expecting 3rd byte between 80 and BF.";
              // *faulty_bytes = 3;
              return false;
            }
          } else {
          // *message = "After a first byte of ED, expecting two following bytes.";
          // *faulty_bytes = 1;
          return false;
        }

        i += 3;
      } else if (u8a[i] >= 0xEE && u8a[i] <= 0xEF)
      /* EE..EF 80..BF 80..BF */
      {
        if (i + 2 < len)
          /* Expect a 2nd and 3rd byte */
          {
            if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0xBF) {
              // *message = "After a first byte between EE and EF, expecting 2nd byte between 80 and BF.";
              // *faulty_bytes = 2;
              return false;
            }

            if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
              // *message = "After a first byte between EE and EF, expecting 3rd byte between 80 and BF.";
              // *faulty_bytes = 3;
              return false;
            }
          } else {
          // *message = "After a first byte between EE and EF, two following bytes.";
          // *faulty_bytes = 1;
          return false;
        }

        i += 3;
      } else if (u8a[i] === 0xF0)
      /* F0 90..BF 80..BF 80..BF */
      {
        if (i + 3 < len)
          /* Expect a 2nd, 3rd 3th byte */
          {
            if (u8a[i + 1] < 0x90 || u8a[i + 1] > 0xBF) {
              // *message = "After a first byte of F0, expecting 2nd byte between 90 and BF.";
              // *faulty_bytes = 2;
              return false;
            }

            if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
              // *message = "After a first byte of F0, expecting 3rd byte between 80 and BF.";
              // *faulty_bytes = 3;
              return false;
            }

            if (u8a[i + 3] < 0x80 || u8a[i + 3] > 0xBF) {
              // *message = "After a first byte of F0, expecting 4th byte between 80 and BF.";
              // *faulty_bytes = 4;
              return false;
            }
          } else {
          // *message = "After a first byte of F0, expecting three following bytes.";
          // *faulty_bytes = 1;
          return false;
        }

        i += 4;
      } else if (u8a[i] >= 0xF1 && u8a[i] <= 0xF3)
      /* F1..F3 80..BF 80..BF 80..BF */
      {
        if (i + 3 < len)
          /* Expect a 2nd, 3rd 3th byte */
          {
            if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0xBF) {
              // *message = "After a first byte of F1, F2, or F3, expecting a 2nd byte between 80 and BF.";
              // *faulty_bytes = 2;
              return false;
            }

            if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
              // *message = "After a first byte of F1, F2, or F3, expecting a 3rd byte between 80 and BF.";
              // *faulty_bytes = 3;
              return false;
            }

            if (u8a[i + 3] < 0x80 || u8a[i + 3] > 0xBF) {
              // *message = "After a first byte of F1, F2, or F3, expecting a 4th byte between 80 and BF.";
              // *faulty_bytes = 4;
              return false;
            }
          } else {
          // *message = "After a first byte of F1, F2, or F3, expecting three following bytes.";
          // *faulty_bytes = 1;
          return false;
        }

        i += 4;
      } else if (u8a[i] === 0xF4)
      /* F4 80..8F 80..BF 80..BF */
      {
        if (i + 3 < len)
          /* Expect a 2nd, 3rd 3th byte */
          {
            if (u8a[i + 1] < 0x80 || u8a[i + 1] > 0x8F) {
              // *message = "After a first byte of F4, expecting 2nd byte between 80 and 8F.";
              // *faulty_bytes = 2;
              return false;
            }

            if (u8a[i + 2] < 0x80 || u8a[i + 2] > 0xBF) {
              // *message = "After a first byte of F4, expecting 3rd byte between 80 and BF.";
              // *faulty_bytes = 3;
              return false;
            }

            if (u8a[i + 3] < 0x80 || u8a[i + 3] > 0xBF) {
              // *message = "After a first byte of F4, expecting 4th byte between 80 and BF.";
              // *faulty_bytes = 4;
              return false;
            }
          } else {
          // *message = "After a first byte of F4, expecting three following bytes.";
          // *faulty_bytes = 1;
          return false;
        }

        i += 4;
      } else {
      // *message = "Expecting bytes in the following ranges: 00..7F C2..F4.";
      // *faulty_bytes = 1;
      return false;
    }
  }

  return true;
}