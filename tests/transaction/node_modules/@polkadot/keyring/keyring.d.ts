import type { EncryptedJsonEncoding, Keypair, KeypairType } from '@polkadot/util-crypto/types';
import type { KeyringInstance, KeyringOptions, KeyringPair, KeyringPair$Json, KeyringPair$Meta } from './types';
import { decodeAddress } from '@polkadot/util-crypto';
/**
 * # @polkadot/keyring
 *
 * ## Overview
 *
 * @name Keyring
 * @summary Keyring management of user accounts
 * @description Allows generation of keyring pairs from a variety of input combinations, such as
 * json object containing account address or public key, account metadata, and account encoded using
 * `addFromJson`, or by providing those values as arguments separately to `addFromAddress`,
 * or by providing the mnemonic (seed phrase) and account metadata as arguments to `addFromMnemonic`.
 * Stores the keyring pairs in a keyring pair dictionary. Removal of the keyring pairs from the keyring pair
 * dictionary is achieved using `removePair`. Retrieval of all the stored pairs via `getPairs` or perform
 * lookup of a pair for a given account address or public key using `getPair`. JSON metadata associated with
 * an account may be obtained using `toJson` accompanied by the account passphrase.
 */
export declare class Keyring implements KeyringInstance {
    #private;
    decodeAddress: typeof decodeAddress;
    constructor(options?: KeyringOptions);
    /**
     * @description retrieve the pairs (alias for getPairs)
     */
    get pairs(): KeyringPair[];
    /**
     * @description retrieve the publicKeys (alias for getPublicKeys)
     */
    get publicKeys(): Uint8Array[];
    /**
     * @description Returns the type of the keyring, ed25519, sr25519 or ecdsa
     */
    get type(): KeypairType;
    /**
     * @name addPair
     * @summary Stores an account, given a keyring pair, as a Key/Value (public key, pair) in Keyring Pair Dictionary
     */
    addPair(pair: KeyringPair): KeyringPair;
    /**
     * @name addFromAddress
     * @summary Stores an account, given an account address, as a Key/Value (public key, pair) in Keyring Pair Dictionary
     * @description Allows user to explicitly provide separate inputs including account address or public key, and optionally
     * the associated account metadata, and the default encoded value as arguments (that may be obtained from the json file
     * of an account backup), and then generates a keyring pair from them that it passes to
     * `addPair` to stores in a keyring pair dictionary the public key of the generated pair as a key and the pair as the associated value.
     */
    addFromAddress(address: string | Uint8Array, meta?: KeyringPair$Meta, encoded?: Uint8Array | null, type?: KeypairType, ignoreChecksum?: boolean, encType?: EncryptedJsonEncoding[]): KeyringPair;
    /**
     * @name addFromJson
     * @summary Stores an account, given JSON data, as a Key/Value (public key, pair) in Keyring Pair Dictionary
     * @description Allows user to provide a json object argument that contains account information (that may be obtained from the json file
     * of an account backup), and then generates a keyring pair from it that it passes to
     * `addPair` to stores in a keyring pair dictionary the public key of the generated pair as a key and the pair as the associated value.
     */
    addFromJson(json: KeyringPair$Json, ignoreChecksum?: boolean): KeyringPair;
    /**
     * @name addFromMnemonic
     * @summary Stores an account, given a mnemonic, as a Key/Value (public key, pair) in Keyring Pair Dictionary
     * @description Allows user to provide a mnemonic (seed phrase that is provided when account is originally created)
     * argument and a metadata argument that contains account information (that may be obtained from the json file
     * of an account backup), and then generates a keyring pair from it that it passes to
     * `addPair` to stores in a keyring pair dictionary the public key of the generated pair as a key and the pair as the associated value.
     */
    addFromMnemonic(mnemonic: string, meta?: KeyringPair$Meta, type?: KeypairType): KeyringPair;
    /**
     * @name addFromPair
     * @summary Stores an account created from an explicit publicKey/secreteKey combination
     */
    addFromPair(pair: Keypair, meta?: KeyringPair$Meta, type?: KeypairType): KeyringPair;
    /**
     * @name addFromSeed
     * @summary Stores an account, given seed data, as a Key/Value (public key, pair) in Keyring Pair Dictionary
     * @description Stores in a keyring pair dictionary the public key of the pair as a key and the pair as the associated value.
     * Allows user to provide the account seed as an argument, and then generates a keyring pair from it that it passes to
     * `addPair` to store in a keyring pair dictionary the public key of the generated pair as a key and the pair as the associated value.
     */
    addFromSeed(seed: Uint8Array, meta?: KeyringPair$Meta, type?: KeypairType): KeyringPair;
    /**
     * @name addFromUri
     * @summary Creates an account via an suri
     * @description Extracts the phrase, path and password from a SURI format for specifying secret keys `<secret>/<soft-key>//<hard-key>///<password>` (the `///password` may be omitted, and `/<soft-key>` and `//<hard-key>` maybe repeated and mixed). The secret can be a hex string, mnemonic phrase or a string (to be padded)
     */
    addFromUri(suri: string, meta?: KeyringPair$Meta, type?: KeypairType): KeyringPair;
    /**
     * @name createFromJson
     * @description Creates a pair from a JSON keyfile
     */
    createFromJson({ address, encoded, encoding: { content, type, version }, meta }: KeyringPair$Json, ignoreChecksum?: boolean): KeyringPair;
    /**
     * @name createFromPair
     * @summary Creates a pair from an explicit publicKey/secreteKey combination
     */
    createFromPair(pair: Keypair, meta?: KeyringPair$Meta, type?: KeypairType): KeyringPair;
    /**
     * @name createFromUri
     * @summary Creates a Keypair from an suri
     * @description This creates a pair from the suri, but does not add it to the keyring
     */
    createFromUri(_suri: string, meta?: KeyringPair$Meta, type?: KeypairType): KeyringPair;
    /**
     * @name encodeAddress
     * @description Encodes the input into an ss58 representation
     */
    encodeAddress: (address: Uint8Array | string, ss58Format?: number | undefined) => string;
    /**
     * @name getPair
     * @summary Retrieves an account keyring pair from the Keyring Pair Dictionary, given an account address
     * @description Returns a keyring pair value from the keyring pair dictionary by performing
     * a key lookup using the provided account address or public key (after decoding it).
     */
    getPair(address: string | Uint8Array): KeyringPair;
    /**
     * @name getPairs
     * @summary Retrieves all account keyring pairs from the Keyring Pair Dictionary
     * @description Returns an array list of all the keyring pair values that are stored in the keyring pair dictionary.
     */
    getPairs(): KeyringPair[];
    /**
     * @name getPublicKeys
     * @summary Retrieves Public Keys of all Keyring Pairs stored in the Keyring Pair Dictionary
     * @description Returns an array list of all the public keys associated with each of the keyring pair values that are stored in the keyring pair dictionary.
     */
    getPublicKeys(): Uint8Array[];
    /**
     * @name removePair
     * @description Deletes the provided input address or public key from the stored Keyring Pair Dictionary.
     */
    removePair(address: string | Uint8Array): void;
    /**
     * @name setSS58Format;
     * @description Sets the ss58 format for the keyring
     */
    setSS58Format(ss58: number): void;
    /**
     * @name toJson
     * @summary Returns a JSON object associated with the input argument that contains metadata assocated with an account
     * @description Returns a JSON object containing the metadata associated with an account
     * when valid address or public key and when the account passphrase is provided if the account secret
     * is not already unlocked and available in memory. Note that in [Polkadot-JS Apps](https://github.com/polkadot-js/apps) the user
     * may backup their account to a JSON file that contains this information.
     */
    toJson(address: string | Uint8Array, passphrase?: string): KeyringPair$Json;
}
