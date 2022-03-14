// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0
const recordIdentity = record => record;

function filterAndApply(events, section, methods, onFound) {
  return events.filter(({
    event
  }) => section === event.section && methods.includes(event.method)).map(record => onFound(record));
}

function getDispatchError({
  event: {
    data: [dispatchError]
  }
}) {
  return dispatchError;
}

function getDispatchInfo({
  event: {
    data,
    method
  }
}) {
  return method === 'ExtrinsicSuccess' ? data[0] : data[1];
}

function extractError(events = []) {
  return filterAndApply(events, 'system', ['ExtrinsicFailed'], getDispatchError)[0];
}

function extractInfo(events = []) {
  return filterAndApply(events, 'system', ['ExtrinsicFailed', 'ExtrinsicSuccess'], getDispatchInfo)[0];
}

export class SubmittableResult {
  constructor({
    dispatchError,
    dispatchInfo,
    events,
    internalError,
    status,
    txHash,
    txIndex
  }) {
    this.dispatchError = dispatchError || extractError(events);
    this.dispatchInfo = dispatchInfo || extractInfo(events);
    this.events = events || [];
    this.internalError = internalError;
    this.status = status;
    this.txHash = txHash;
    this.txIndex = txIndex;
  }

  get isCompleted() {
    return this.isError || this.status.isInBlock || this.status.isFinalized;
  }

  get isError() {
    return this.status.isDropped || this.status.isFinalityTimeout || this.status.isInvalid || this.status.isUsurped;
  }

  get isFinalized() {
    return this.status.isFinalized;
  }

  get isInBlock() {
    return this.status.isInBlock;
  }

  get isWarning() {
    return this.status.isRetracted;
  }
  /**
   * @description Filters EventRecords for the specified method & section (there could be multiple)
   */


  filterRecords(section, method) {
    return filterAndApply(this.events, section, Array.isArray(method) ? method : [method], recordIdentity);
  }
  /**
   * @description Finds an EventRecord for the specified method & section
   */


  findRecord(section, method) {
    return this.filterRecords(section, method)[0];
  }
  /**
   * @description Creates a human representation of the output
   */


  toHuman(isExtended) {
    var _this$dispatchError, _this$dispatchInfo, _this$internalError;

    return {
      dispatchError: (_this$dispatchError = this.dispatchError) === null || _this$dispatchError === void 0 ? void 0 : _this$dispatchError.toHuman(),
      dispatchInfo: (_this$dispatchInfo = this.dispatchInfo) === null || _this$dispatchInfo === void 0 ? void 0 : _this$dispatchInfo.toHuman(),
      events: this.events.map(e => e.toHuman(isExtended)),
      internalError: (_this$internalError = this.internalError) === null || _this$internalError === void 0 ? void 0 : _this$internalError.message.toString(),
      status: this.status.toHuman(isExtended)
    };
  }

}