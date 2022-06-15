// Import
const { ApiPromise, WsProvider } = require('@polkadot/api');
const {Keyring } = require('@polkadot/keyring');

async function main() {
    // Construct

    const wsProvider = new WsProvider('ws://127.0.0.1:8546');
    const api = await ApiPromise.create({ provider: wsProvider });

    // chain defaults
    const genesisHash = await api.genesisHash;
    console.log(`genesis hash: ${genesisHash}`);

    // state_getRuntimeVersion
    const runtimeVersion = await api.rpc.state.getRuntimeVersion();
    console.log('\x1b[32m%s\x1b[0m %s', "runtimeVersion:", runtimeVersion);

    // chain_getBlockHash most current
    const chainGetBlockHashCurrent = await api.rpc.chain.getBlockHash();
    console.log('\x1b[32m%s\x1b[0m %s', 'chainGetBlockHash current:',  chainGetBlockHashCurrent);

    // chain_getBlockHash 0
    const chainGetBlockHash = await api.rpc.chain.getBlockHash(0);
    console.log('\x1b[32m%s\x1b[0m %s', 'chainGetBlockHash 0:',  chainGetBlockHash);

    // chain_getBlockHash 11
    const chainGetBlockHash11 = await api.rpc.chain.getBlockHash(11);
    console.log('\x1b[32m%s\x1b[0m %s', 'chainGetBlockHash 11:',  chainGetBlockHash11);

    // chain_getBlockHash range
    const chainGetBlockHashRange = await api.rpc.chain.getBlockHash([0, 11]);
    console.log('\x1b[32m%s\x1b[0m %s', 'chainGetBlockHash range:',  chainGetBlockHashRange);

    // state_getStorage
    const getStorage = await api.rpc.state.getStorage("0x26aa394eea5630e07c48ae0c9558cef7a44704b568d21667356a5a050c118746e333f8c357e331db45010000");
    console.log('\x1b[32m%s\x1b[0m %s', "getStorage:", getStorage)
    
    // state_queryStorage
    const queryStorage = await api.rpc.state.queryStorage(["0x26aa394eea5630e07c48ae0c9558cef7a44704b568d21667356a5a050c118746e333f8c357e331db45010000"], "0x0a0f4687cfc807af53e28beb2b504c015d1db34e44126e4af9e5489473fe205b", null);
    console.log('\x1b[32m%s\x1b[0m %s', "queryStorage:", queryStorage);
    

    // const runtimeMetadata = await  api.runtimeMetadata;
    // // currently not sending runtimeMetadata to console because it's very large, uncomment if you want to see
    // // console.log(`runtime metadata: ${runtimeMetadata}`);

    // const runtimeVersion = await api.runtimeVersion;
    // console.log(`runtime version: ${runtimeVersion}`);

    // const libraryInfo = await api.libraryInfo;
    // console.log(`library info: ${libraryInfo}`);

    // //Basic queries
    // const now = await api.query.timestamp.now();
    // console.log(`timestamp now: ${now}`);

    // // Retrieve the account balance & nonce via the system module
    // const ADDR_Alice = '5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY';
    // const { nonce, data: balance } = await api.query.system.account(ADDR_Alice);
    // console.log(`Alice: balance of ${balance.free} and a nonce of ${nonce}`)

    // // RPC queries
    // const chain = await api.rpc.system.chain();
    // console.log(`system chain: ${chain}`);

    // const sysProperties = await api.rpc.system.properties();
    // console.log(`system properties: ${sysProperties}`);

    // const chainType = await api.rpc.system.chainType();
    // console.log(`system chainType: ${chainType}`);

    // const header = await api.rpc.chain.getHeader();
    // console.log(`header ${header}`);

    // // Subscribe to the new headers
    // // TODO: Issue: chain.subscribeNewHeads is returning values twice for each result new head.
    // let count = 0;
    // const unsubHeads = await api.rpc.chain.subscribeNewHeads((lastHeader) => {
    //     console.log(`${chain}: last block #${lastHeader.number} has hash ${lastHeader.hash}`);
    //     if (++count === 5) {
    //         unsubHeads();
    //     }
    // });

    // const blockHash = await api.rpc.chain.getBlockHash();
    // console.log(`current blockhash ${blockHash}`);

    // const block = await api.rpc.chain.getBlock(blockHash);
    // console.log(`current block: ${block}`);

    // // Simple transaction
    // // TODO Issue:  This currently fails with error: RPC-CORE: submitExtrinsic(extrinsic: Extrinsic): Hash:: -32000: validator: (nil *modules.Extrinsic): null
    // const keyring = new Keyring({type: 'sr25519' });
    // const aliceKey = keyring.addFromUri('//Alice',  { name: 'Alice default' });
    // console.log(`${aliceKey.meta.name}: has address ${aliceKey.address} with publicKey [${aliceKey.publicKey}]`);

    // const ADDR_Bob = '0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22';

    // const transfer = await api.tx.balances.transfer(ADDR_Bob, 12345)
    //     .signAndSend(aliceKey);

    // console.log(`hxHash ${transfer}`);

}

main().catch(console.error);
