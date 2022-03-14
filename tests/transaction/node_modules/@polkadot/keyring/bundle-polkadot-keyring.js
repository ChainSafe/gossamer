(function (global, factory) {
  typeof exports === 'object' && typeof module !== 'undefined' ? factory(exports, require('@polkadot/util'), require('@polkadot/util-crypto')) :
  typeof define === 'function' && define.amd ? define(['exports', '@polkadot/util', '@polkadot/util-crypto'], factory) :
  (global = typeof globalThis !== 'undefined' ? globalThis : global || self, factory(global.polkadotKeyring = {}, global.polkadotUtil, global.polkadotUtilCrypto));
})(this, (function (exports, util, utilCrypto) { 'use strict';

  const global = window;

  const DEV_PHRASE = 'bottom drive obey lake curtain smoke basket hold race lonely fit walk';
  const DEV_SEED = '0xfac7959dbfe72f052e5a0c3c8d6530f202b02fd8f9f5ca3580ec8deb7797479e';

  const PKCS8_DIVIDER = new Uint8Array([161, 35, 3, 33, 0]);
  const PKCS8_HEADER = new Uint8Array([48, 83, 2, 1, 1, 48, 5, 6, 3, 43, 101, 112, 4, 34, 4, 32]);
  const PUB_LENGTH = 32;
  const SEC_LENGTH = 64;
  const SEED_LENGTH = 32;

  const SEED_OFFSET = PKCS8_HEADER.length;
  function decodePair(passphrase, encrypted, _encType) {
    const encType = Array.isArray(_encType) || util.isUndefined(_encType) ? _encType : [_encType];
    const decrypted = utilCrypto.jsonDecryptData(encrypted, passphrase, encType);
    const header = decrypted.subarray(0, PKCS8_HEADER.length);
    util.assert(util.u8aEq(header, PKCS8_HEADER), 'Invalid Pkcs8 header found in body');
    let secretKey = decrypted.subarray(SEED_OFFSET, SEED_OFFSET + SEC_LENGTH);
    let divOffset = SEED_OFFSET + SEC_LENGTH;
    let divider = decrypted.subarray(divOffset, divOffset + PKCS8_DIVIDER.length);
    if (!util.u8aEq(divider, PKCS8_DIVIDER)) {
      divOffset = SEED_OFFSET + SEED_LENGTH;
      secretKey = decrypted.subarray(SEED_OFFSET, divOffset);
      divider = decrypted.subarray(divOffset, divOffset + PKCS8_DIVIDER.length);
      util.assert(util.u8aEq(divider, PKCS8_DIVIDER), 'Invalid Pkcs8 divider found in body');
    }
    const pubOffset = divOffset + PKCS8_DIVIDER.length;
    const publicKey = decrypted.subarray(pubOffset, pubOffset + PUB_LENGTH);
    return {
      publicKey,
      secretKey
    };
  }

  function encodePair({
    publicKey,
    secretKey
  }, passphrase) {
    util.assert(secretKey, 'Expected a valid secretKey to be passed to encode');
    const encoded = util.u8aConcat(PKCS8_HEADER, secretKey, PKCS8_DIVIDER, publicKey);
    if (!passphrase) {
      return encoded;
    }
    const {
      params,
      password,
      salt
    } = utilCrypto.scryptEncode(passphrase);
    const {
      encrypted,
      nonce
    } = utilCrypto.naclEncrypt(encoded, password.subarray(0, 32));
    return util.u8aConcat(utilCrypto.scryptToU8a(salt, params), nonce, encrypted);
  }

  function pairToJson(type, {
    address,
    meta
  }, encoded, isEncrypted) {
    return util.objectSpread(utilCrypto.jsonEncryptFormat(encoded, ['pkcs8', type], isEncrypted), {
      address,
      meta
    });
  }

  const SIG_TYPE_NONE = new Uint8Array();
  const TYPE_FROM_SEED = {
    ecdsa: utilCrypto.secp256k1PairFromSeed,
    ed25519: utilCrypto.ed25519PairFromSeed,
    ethereum: utilCrypto.secp256k1PairFromSeed,
    sr25519: utilCrypto.sr25519PairFromSeed
  };
  const TYPE_PREFIX = {
    ecdsa: new Uint8Array([2]),
    ed25519: new Uint8Array([0]),
    ethereum: new Uint8Array([2]),
    sr25519: new Uint8Array([1])
  };
  const TYPE_SIGNATURE = {
    ecdsa: (m, p) => utilCrypto.secp256k1Sign(m, p, 'blake2'),
    ed25519: utilCrypto.ed25519Sign,
    ethereum: (m, p) => utilCrypto.secp256k1Sign(m, p, 'keccak'),
    sr25519: utilCrypto.sr25519Sign
  };
  const TYPE_ADDRESS = {
    ecdsa: p => p.length > 32 ? utilCrypto.blake2AsU8a(p) : p,
    ed25519: p => p,
    ethereum: p => p.length === 20 ? p : utilCrypto.keccakAsU8a(utilCrypto.secp256k1Expand(p)),
    sr25519: p => p
  };
  function isLocked(secretKey) {
    return !secretKey || util.u8aEmpty(secretKey);
  }
  function vrfHash(proof, context, extra) {
    return utilCrypto.blake2AsU8a(util.u8aConcat(context || '', extra || '', proof));
  }
  function createPair({
    toSS58,
    type
  }, {
    publicKey,
    secretKey
  }, meta = {}, encoded = null, encTypes) {
    const decodePkcs8 = (passphrase, userEncoded) => {
      const decoded = decodePair(passphrase, userEncoded || encoded, encTypes);
      if (decoded.secretKey.length === 64) {
        publicKey = decoded.publicKey;
        secretKey = decoded.secretKey;
      } else {
        const pair = TYPE_FROM_SEED[type](decoded.secretKey);
        publicKey = pair.publicKey;
        secretKey = pair.secretKey;
      }
    };
    const recode = passphrase => {
      isLocked(secretKey) && encoded && decodePkcs8(passphrase, encoded);
      encoded = encodePair({
        publicKey,
        secretKey
      }, passphrase);
      encTypes = undefined;
      return encoded;
    };
    const encodeAddress = () => {
      const raw = TYPE_ADDRESS[type](publicKey);
      return type === 'ethereum' ? utilCrypto.ethereumEncode(raw) : toSS58(raw);
    };
    return {
      get address() {
        return encodeAddress();
      },
      get addressRaw() {
        const raw = TYPE_ADDRESS[type](publicKey);
        return type === 'ethereum' ? raw.slice(-20) : raw;
      },
      get isLocked() {
        return isLocked(secretKey);
      },
      get meta() {
        return meta;
      },
      get publicKey() {
        return publicKey;
      },
      get type() {
        return type;
      },
      decodePkcs8,
      decryptMessage: (encryptedMessageWithNonce, senderPublicKey) => {
        util.assert(!isLocked(secretKey), 'Cannot encrypt with a locked key pair');
        util.assert(!['ecdsa', 'ethereum'].includes(type), 'Secp256k1 not supported yet');
        const messageU8a = util.u8aToU8a(encryptedMessageWithNonce);
        return utilCrypto.naclOpen(messageU8a.slice(24, messageU8a.length), messageU8a.slice(0, 24), utilCrypto.convertPublicKeyToCurve25519(util.u8aToU8a(senderPublicKey)), utilCrypto.convertSecretKeyToCurve25519(secretKey));
      },
      derive: (suri, meta) => {
        util.assert(type !== 'ethereum', 'Unable to derive on this keypair');
        util.assert(!isLocked(secretKey), 'Cannot derive on a locked keypair');
        const {
          path
        } = utilCrypto.keyExtractPath(suri);
        const derived = utilCrypto.keyFromPath({
          publicKey,
          secretKey
        }, path, type);
        return createPair({
          toSS58,
          type
        }, derived, meta, null);
      },
      encodePkcs8: passphrase => {
        return recode(passphrase);
      },
      encryptMessage: (message, recipientPublicKey, nonceIn) => {
        util.assert(!isLocked(secretKey), 'Cannot encrypt with a locked key pair');
        util.assert(!['ecdsa', 'ethereum'].includes(type), 'Secp256k1 not supported yet');
        const {
          nonce,
          sealed
        } = utilCrypto.naclSeal(util.u8aToU8a(message), utilCrypto.convertSecretKeyToCurve25519(secretKey), utilCrypto.convertPublicKeyToCurve25519(util.u8aToU8a(recipientPublicKey)), nonceIn);
        return util.u8aConcat(nonce, sealed);
      },
      lock: () => {
        secretKey = new Uint8Array();
      },
      setMeta: additional => {
        meta = util.objectSpread({}, meta, additional);
      },
      sign: (message, options = {}) => {
        util.assert(!isLocked(secretKey), 'Cannot sign with a locked key pair');
        return util.u8aConcat(options.withType ? TYPE_PREFIX[type] : SIG_TYPE_NONE, TYPE_SIGNATURE[type](util.u8aToU8a(message), {
          publicKey,
          secretKey
        }));
      },
      toJson: passphrase => {
        const address = ['ecdsa', 'ethereum'].includes(type) ? publicKey.length === 20 ? util.u8aToHex(publicKey) : util.u8aToHex(utilCrypto.secp256k1Compress(publicKey)) : encodeAddress();
        return pairToJson(type, {
          address,
          meta
        }, recode(passphrase), !!passphrase);
      },
      unlock: passphrase => {
        return decodePkcs8(passphrase);
      },
      verify: (message, signature, signerPublic) => {
        return utilCrypto.signatureVerify(message, signature, TYPE_ADDRESS[type](util.u8aToU8a(signerPublic))).isValid;
      },
      vrfSign: (message, context, extra) => {
        util.assert(!isLocked(secretKey), 'Cannot sign with a locked key pair');
        if (type === 'sr25519') {
          return utilCrypto.sr25519VrfSign(message, {
            secretKey
          }, context, extra);
        }
        const proof = TYPE_SIGNATURE[type](util.u8aToU8a(message), {
          publicKey,
          secretKey
        });
        return util.u8aConcat(vrfHash(proof, context, extra), proof);
      },
      vrfVerify: (message, vrfResult, signerPublic, context, extra) => {
        if (type === 'sr25519') {
          return utilCrypto.sr25519VrfVerify(message, vrfResult, publicKey, context, extra);
        }
        const result = utilCrypto.signatureVerify(message, util.u8aConcat(TYPE_PREFIX[type], vrfResult.subarray(32)), TYPE_ADDRESS[type](util.u8aToU8a(signerPublic)));
        return result.isValid && util.u8aEq(vrfResult.subarray(0, 32), vrfHash(vrfResult.subarray(32), context, extra));
      }
    };
  }

  class Pairs {
    #map = {};
    add(pair) {
      this.#map[utilCrypto.decodeAddress(pair.address).toString()] = pair;
      return pair;
    }
    all() {
      return Object.values(this.#map);
    }
    get(address) {
      const pair = this.#map[utilCrypto.decodeAddress(address).toString()];
      util.assert(pair, () => `Unable to retrieve keypair '${util.isU8a(address) || util.isHex(address) ? util.u8aToHex(util.u8aToU8a(address)) : address}'`);
      return pair;
    }
    remove(address) {
      delete this.#map[utilCrypto.decodeAddress(address).toString()];
    }
  }

  const PairFromSeed = {
    ecdsa: seed => utilCrypto.secp256k1PairFromSeed(seed),
    ed25519: seed => utilCrypto.ed25519PairFromSeed(seed),
    ethereum: seed => utilCrypto.secp256k1PairFromSeed(seed),
    sr25519: seed => utilCrypto.sr25519PairFromSeed(seed)
  };
  function pairToPublic({
    publicKey
  }) {
    return publicKey;
  }
  class Keyring {
    #pairs;
    #type;
    #ss58;
    decodeAddress = utilCrypto.decodeAddress;
    constructor(options = {}) {
      options.type = options.type || 'ed25519';
      util.assert(['ecdsa', 'ethereum', 'ed25519', 'sr25519'].includes(options.type || 'undefined'), () => `Expected a keyring type of either 'ed25519', 'sr25519', 'ethereum' or 'ecdsa', found '${options.type || 'unknown'}`);
      this.#pairs = new Pairs();
      this.#ss58 = options.ss58Format;
      this.#type = options.type;
    }
    get pairs() {
      return this.getPairs();
    }
    get publicKeys() {
      return this.getPublicKeys();
    }
    get type() {
      return this.#type;
    }
    addPair(pair) {
      return this.#pairs.add(pair);
    }
    addFromAddress(address, meta = {}, encoded = null, type = this.type, ignoreChecksum, encType) {
      const publicKey = this.decodeAddress(address, ignoreChecksum);
      return this.addPair(createPair({
        toSS58: this.encodeAddress,
        type
      }, {
        publicKey,
        secretKey: new Uint8Array()
      }, meta, encoded, encType));
    }
    addFromJson(json, ignoreChecksum) {
      return this.addPair(this.createFromJson(json, ignoreChecksum));
    }
    addFromMnemonic(mnemonic, meta = {}, type = this.type) {
      return this.addFromUri(mnemonic, meta, type);
    }
    addFromPair(pair, meta = {}, type = this.type) {
      return this.addPair(this.createFromPair(pair, meta, type));
    }
    addFromSeed(seed, meta = {}, type = this.type) {
      return this.addPair(createPair({
        toSS58: this.encodeAddress,
        type
      }, PairFromSeed[type](seed), meta, null));
    }
    addFromUri(suri, meta = {}, type = this.type) {
      return this.addPair(this.createFromUri(suri, meta, type));
    }
    createFromJson({
      address,
      encoded,
      encoding: {
        content,
        type,
        version
      },
      meta
    }, ignoreChecksum) {
      util.assert(version !== '3' || content[0] === 'pkcs8', () => `Unable to decode non-pkcs8 type, [${content.join(',')}] found}`);
      const cryptoType = version === '0' || !Array.isArray(content) ? this.type : content[1];
      const encType = !Array.isArray(type) ? [type] : type;
      util.assert(['ed25519', 'sr25519', 'ecdsa', 'ethereum'].includes(cryptoType), () => `Unknown crypto type ${cryptoType}`);
      const publicKey = util.isHex(address) ? util.hexToU8a(address) : this.decodeAddress(address, ignoreChecksum);
      const decoded = util.isHex(encoded) ? util.hexToU8a(encoded) : utilCrypto.base64Decode(encoded);
      return createPair({
        toSS58: this.encodeAddress,
        type: cryptoType
      }, {
        publicKey,
        secretKey: new Uint8Array()
      }, meta, decoded, encType);
    }
    createFromPair(pair, meta = {}, type = this.type) {
      return createPair({
        toSS58: this.encodeAddress,
        type
      }, pair, meta, null);
    }
    createFromUri(_suri, meta = {}, type = this.type) {
      const suri = _suri.startsWith('//') ? `${DEV_PHRASE}${_suri}` : _suri;
      const {
        derivePath,
        password,
        path,
        phrase
      } = utilCrypto.keyExtractSuri(suri);
      let seed;
      const isPhraseHex = util.isHex(phrase, 256);
      if (isPhraseHex) {
        seed = util.hexToU8a(phrase);
      } else {
        const parts = phrase.split(' ');
        if ([12, 15, 18, 21, 24].includes(parts.length)) {
          seed = type === 'ethereum' ? utilCrypto.mnemonicToLegacySeed(phrase, '', false, 64) : utilCrypto.mnemonicToMiniSecret(phrase, password);
        } else {
          util.assert(phrase.length <= 32, 'specified phrase is not a valid mnemonic and is invalid as a raw seed at > 32 bytes');
          seed = util.stringToU8a(phrase.padEnd(32));
        }
      }
      const derived = type === 'ethereum' ? isPhraseHex ? PairFromSeed[type](seed)
      : utilCrypto.hdEthereum(seed, derivePath.substring(1)) : utilCrypto.keyFromPath(PairFromSeed[type](seed), path, type);
      return createPair({
        toSS58: this.encodeAddress,
        type
      }, derived, meta, null);
    }
    encodeAddress = (address, ss58Format) => {
      return this.type === 'ethereum' ? utilCrypto.ethereumEncode(address) : utilCrypto.encodeAddress(address, util.isUndefined(ss58Format) ? this.#ss58 : ss58Format);
    };
    getPair(address) {
      return this.#pairs.get(address);
    }
    getPairs() {
      return this.#pairs.all();
    }
    getPublicKeys() {
      return this.#pairs.all().map(pairToPublic);
    }
    removePair(address) {
      this.#pairs.remove(address);
    }
    setSS58Format(ss58) {
      this.#ss58 = ss58;
    }
    toJson(address, passphrase) {
      return this.#pairs.get(address).toJson(passphrase);
    }
  }

  const packageInfo = {
    name: '@polkadot/keyring',
    path: new URL('.', (typeof document === 'undefined' && typeof location === 'undefined' ? new (require('u' + 'rl').URL)('file:' + __filename).href : typeof document === 'undefined' ? location.href : (document.currentScript && document.currentScript.src || new URL('bundle-polkadot-keyring.js', document.baseURI).href))).pathname,
    type: 'esm',
    version: '8.4.1'
  };

  const PAIRSSR25519 = [{
    publicKey: util.hexToU8a('0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d'),
    secretKey: util.hexToU8a('0x98319d4ff8a9508c4bb0cf0b5a78d760a0b2082c02775e6e82370816fedfff48925a225d97aa00682d6a59b95b18780c10d7032336e88f3442b42361f4a66011'),
    seed: 'Alice',
    type: 'sr25519'
  }, {
    publicKey: util.hexToU8a('0xbe5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f'),
    secretKey: util.hexToU8a('0xe8da6c9d810e020f5e3c7f5af2dea314cbeaa0d72bc6421e92c0808a0c584a6046ab28e97c3ffc77fe12b5a4d37e8cd4afbfebbf2391ffc7cb07c0f38c023efd'),
    seed: 'Alice//stash',
    type: 'sr25519'
  }, {
    publicKey: util.hexToU8a('0x8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48'),
    secretKey: util.hexToU8a('0x081ff694633e255136bdb456c20a5fc8fed21f8b964c11bb17ff534ce80ebd5941ae88f85d0c1bfc37be41c904e1dfc01de8c8067b0d6d5df25dd1ac0894a325'),
    seed: 'Bob',
    type: 'sr25519'
  }, {
    publicKey: util.hexToU8a('0xfe65717dad0447d715f660a0a58411de509b42e6efb8375f562f58a554d5860e'),
    secretKey: util.hexToU8a('0xc006507cdfc267a21532394c49ca9b754ca71de21e15a1cdf807c7ceab6d0b6c3ed408d9d35311540dcd54931933e67cf1ea10d46f75408f82b789d9bd212fde'),
    seed: 'Bob//stash',
    type: 'sr25519'
  }, {
    publicKey: util.hexToU8a('0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22'),
    secretKey: util.hexToU8a('0xa8f2d83016052e5d6d77b2f6fd5d59418922a09024cda701b3c34369ec43a7668faf12ff39cd4e5d92bb773972f41a7a5279ebc2ed92264bed8f47d344f8f18c'),
    seed: 'Charlie',
    type: 'sr25519'
  }, {
    publicKey: util.hexToU8a('0x306721211d5404bd9da88e0204360a1a9ab8b87c66c1bc2fcdd37f3c2222cc20'),
    secretKey: util.hexToU8a('0x20e05482ca4677e0edbc58ae9a3a59f6ed3b1a9484ba17e64d6fe8688b2b7b5d108c4487b9323b98b11fe36cb301b084e920f7b7895536809a6d62a451b25568'),
    seed: 'Dave',
    type: 'sr25519'
  }, {
    publicKey: util.hexToU8a('0xe659a7a1628cdd93febc04a4e0646ea20e9f5f0ce097d9a05290d4a9e054df4e'),
    secretKey: util.hexToU8a('0x683576abfd5dc35273e4264c23095a1bf21c14517bece57c7f0cc5c0ed4ce06a3dbf386b7828f348abe15d76973a72009e6ef86a5c91db2990cb36bb657c6587'),
    seed: 'Eve',
    type: 'sr25519'
  }, {
    publicKey: util.hexToU8a('0x1cbd2d43530a44705ad088af313e18f80b53ef16b36177cd4b77b846f2a5f07c'),
    secretKey: util.hexToU8a('0xb835c20f450079cf4f513900ae9faf8df06ad86c681884122c752a4b2bf74d4303e4f21bc6cc62bb4eeed5a9cce642c25e2d2ac1464093b50f6196d78e3a7426'),
    seed: 'Ferdie',
    type: 'sr25519'
  }];
  const PAIRSETHEREUM = [{
    name: 'Alith',
    publicKey: util.hexToU8a('0x02509540919faacf9ab52146c9aa40db68172d83777250b28e4679176e49ccdd9f'),
    secretKey: util.hexToU8a('0x5fb92d6e98884f76de468fa3f6278f8807c48bebc13595d45af5bdc4da702133'),
    type: 'ethereum'
  }, {
    name: 'Baltathar',
    publicKey: util.hexToU8a('0x033bc19e36ff1673910575b6727a974a9abd80c9a875d41ab3e2648dbfb9e4b518'),
    secretKey: util.hexToU8a('0x8075991ce870b93a8870eca0c0f91913d12f47948ca0fd25b49c6fa7cdbeee8b'),
    type: 'ethereum'
  }, {
    name: 'Charleth',
    publicKey: util.hexToU8a('0x0234637bdc0e89b5d46543bcbf8edff329d2702bc995e27e9af4b1ba009a3c2a5e'),
    secretKey: util.hexToU8a('0x0b6e18cafb6ed99687ec547bd28139cafdd2bffe70e6b688025de6b445aa5c5b'),
    type: 'ethereum'
  }, {
    name: 'Dorothy',
    publicKey: util.hexToU8a('0x02a00d60b2b408c2a14c5d70cdd2c205db8985ef737a7e55ad20ea32cc9e7c417c'),
    secretKey: util.hexToU8a('0x39539ab1876910bbf3a223d84a29e28f1cb4e2e456503e7e91ed39b2e7223d68'),
    type: 'ethereum'
  }, {
    name: 'Ethan',
    publicKey: util.hexToU8a('0x025cdc005b752651cd3f728fb9192182acb3a9c89e19072cbd5b03f3ee1f1b3ffa'),
    secretKey: util.hexToU8a('0x7dce9bc8babb68fec1409be38c8e1a52650206a7ed90ff956ae8a6d15eeaaef4'),
    type: 'ethereum'
  }, {
    name: 'Faith',
    publicKey: util.hexToU8a('0x037964b6c9d546da4646ada28a99e34acaa1d14e7aba861a9055f9bd200c8abf74'),
    secretKey: util.hexToU8a('0xb9d2ea9a615f3165812e8d44de0d24da9bbd164b65c4f0573e1ce2c8dbd9c8df'),
    type: 'ethereum'
  }];
  function createMeta(name, seed) {
    util.assert(name || seed, 'Testing pair should have either a name or a seed');
    return {
      isTesting: true,
      name: name || seed && seed.replace('//', '_').toLowerCase()
    };
  }
  function createTestKeyring(options = {}, isDerived = true) {
    const keyring = new Keyring(options);
    const pairs = options.type && options.type === 'ethereum' ? PAIRSETHEREUM : PAIRSSR25519;
    for (const {
      name,
      publicKey,
      secretKey,
      seed,
      type
    } of pairs) {
      const meta = createMeta(name, seed);
      const pair = !isDerived && !name && seed ? keyring.addFromUri(seed, meta, options.type) : keyring.addPair(createPair({
        toSS58: keyring.encodeAddress,
        type
      }, {
        publicKey,
        secretKey
      }, meta));
      pair.lock = () => {
      };
    }
    return keyring;
  }

  const publicKey = new Uint8Array(32);
  const address = utilCrypto.encodeAddress(publicKey);
  const meta = {
    isTesting: true,
    name: 'nobody'
  };
  const json = {
    address,
    encoded: '',
    encoding: {
      content: ['pkcs8', 'ed25519'],
      type: 'none',
      version: '0'
    },
    meta
  };
  const pair = {
    address,
    addressRaw: publicKey,
    decodePkcs8: (passphrase, encoded) => undefined,
    decryptMessage: (encryptedMessageWithNonce, senderPublicKey) => null,
    derive: (suri, meta) => pair,
    encodePkcs8: passphrase => new Uint8Array(0),
    encryptMessage: (message, recipientPublicKey, _nonce) => new Uint8Array(),
    isLocked: true,
    lock: () => {
    },
    meta,
    publicKey,
    setMeta: meta => undefined,
    sign: message => new Uint8Array(64),
    toJson: passphrase => json,
    type: 'ed25519',
    unlock: passphrase => undefined,
    verify: (message, signature) => false,
    vrfSign: (message, context, extra) => new Uint8Array(96),
    vrfVerify: (message, vrfResult, context, extra) => false
  };
  function nobody() {
    return pair;
  }

  function createTestPairs(options, isDerived = true) {
    const keyring = createTestKeyring(options, isDerived);
    const pairs = keyring.getPairs();
    const map = {
      nobody: nobody()
    };
    for (const p of pairs) {
      map[p.meta.name] = p;
    }
    return map;
  }

  Object.defineProperty(exports, 'decodeAddress', {
    enumerable: true,
    get: function () { return utilCrypto.decodeAddress; }
  });
  Object.defineProperty(exports, 'encodeAddress', {
    enumerable: true,
    get: function () { return utilCrypto.encodeAddress; }
  });
  Object.defineProperty(exports, 'setSS58Format', {
    enumerable: true,
    get: function () { return utilCrypto.setSS58Format; }
  });
  exports.DEV_PHRASE = DEV_PHRASE;
  exports.DEV_SEED = DEV_SEED;
  exports.Keyring = Keyring;
  exports.createPair = createPair;
  exports.createTestKeyring = createTestKeyring;
  exports.createTestPairs = createTestPairs;
  exports.packageInfo = packageInfo;

  Object.defineProperty(exports, '__esModule', { value: true });

}));
