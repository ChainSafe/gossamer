import type { KeyringOptions, KeyringPair } from './types';
export interface TestKeyringMap {
    [index: string]: KeyringPair;
}
export declare function createTestPairs(options?: KeyringOptions, isDerived?: boolean): TestKeyringMap;
