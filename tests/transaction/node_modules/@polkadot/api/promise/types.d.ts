import type { SubmittableExtrinsic as SubmittableExtrinsicBase } from '../submittable/types';
import type { QueryableStorageEntry as QueryableStorageEntryBase, SubmittableExtrinsicFunction as SubmittableExtrinsicFunctionBase } from '../types';
export declare type QueryableStorageEntry = QueryableStorageEntryBase<'promise'>;
export declare type SubmittableExtrinsic = SubmittableExtrinsicBase<'promise'>;
export declare type SubmittableExtrinsicFunction = SubmittableExtrinsicFunctionBase<'promise'>;
