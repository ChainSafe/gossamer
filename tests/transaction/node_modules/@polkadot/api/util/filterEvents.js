// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { l } from "./logging.js";
export function filterEvents(txHash, {
  block: {
    extrinsics,
    header
  }
}, allEvents, status) {
  // extrinsics to hashes
  for (const [txIndex, x] of extrinsics.entries()) {
    if (x.hash.eq(txHash)) {
      return {
        events: allEvents.filter(({
          phase
        }) => phase.isApplyExtrinsic && phase.asApplyExtrinsic.eqn(txIndex)),
        txIndex
      };
    }
  } // if we do get the block after finalized, it _should_ be there
  // only warn on filtering with isInBlock (finalization finalizes after)


  if (status.isInBlock) {
    const allHashes = extrinsics.map(x => x.hash.toHex());
    l.warn(`block ${header.hash.toHex()}: Unable to find extrinsic ${txHash.toHex()} inside ${allHashes.join(', ')}`);
  }

  return {};
}