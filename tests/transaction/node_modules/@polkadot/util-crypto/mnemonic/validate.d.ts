/**
 * @name mnemonicValidate
 * @summary Validates a mnemonic input using [BIP39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki).
 * @example
 * <BR>
 *
 * ```javascript
 * import { mnemonicGenerate, mnemonicValidate } from '@polkadot/util-crypto';
 *
 * const mnemonic = mnemonicGenerate(); // => string
 * const isValidMnemonic = mnemonicValidate(mnemonic); // => boolean
 * ```
 */
export declare function mnemonicValidate(mnemonic: string, onlyJs?: boolean): boolean;
