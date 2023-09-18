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

    const runtimeMetadata = await  api.runtimeMetadata;
    // currently not sending runtimeMetadata to console because it's very large, uncomment if you want to see
    // console.log(`runtime metadata: ${runtimeMetadata}`);

    const runtimeVersion = await api.runtimeVersion;
    console.log(`runtime version: ${runtimeVersion}`);

    const libraryInfo = await api.libraryInfo;
    console.log(`library info: ${libraryInfo}`);

    //Basic queries
    const now = await api.query.timestamp.now();
    console.log(`timestamp now: ${now}`);

    // Retrieve the account balance & nonce via the system module
    const ADDR_Alice = '5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY';
    const { nonce, data: balance } = await api.query.system.account(ADDR_Alice);
    console.log(`Alice: balance of ${balance.free} and a nonce of ${nonce}`)

    // RPC queries
    const chain = await api.rpc.system.chain();
    console.log(`system chain: ${chain}`);

    const sysProperties = await api.rpc.system.properties();
    console.log(`system properties: ${sysProperties}`);

    const chainType = await api.rpc.system.chainType();
    console.log(`system chainType: ${chainType}`);

    const header = await api.rpc.chain.getHeader();
    console.log(`header ${header}`);

    // Subscribe to the new headers
    // TODO: Issue: chain.subscribeNewHeads is returning values twice for each result new head.
    let count = 0;
    const unsubHeads = await api.rpc.chain.subscribeNewHeads((lastHeader) => {
        console.log(`${chain}: last block #${lastHeader.number} has hash ${lastHeader.hash}`);
        if (++count === 5) {
            unsubHeads();
        }
    });

    const blockHash = await api.rpc.chain.getBlockHash();
    console.log(`current blockhash ${blockHash}`);

    const block = await api.rpc.chain.getBlock(blockHash);
    console.log(`current block: ${block}`);

    // Simple transaction
    // TODO Issue:  This currently fails with error: RPC-CORE: submitExtrinsic(extrinsic: Extrinsic): Hash:: -32000: validator: (nil *modules.Extrinsic): null
    const keyring = new Keyring({type: 'sr25519' });
    const aliceKey = keyring.addFromUri('//Alice',  { name: 'Alice default' });
    console.log(`${aliceKey.meta.name}: has address ${aliceKey.address} with publicKey [${aliceKey.publicKey}]`);

    const ADDR_Charlie = '0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22';

    const transfer = await api.tx.balances.transfer(ADDR_Charlie, 12345)
        .signAndSend(aliceKey);

    console.log(`hxHash ${transfer}`);

}

main().catch(console.error);
