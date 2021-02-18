// Import
const { ApiPromise, WsProvider } = require('@polkadot/api');
const {Keyring } = require('@polkadot/keyring');

async function main() {
    // Construct

    const wsProvider = new WsProvider('ws://127.0.0.1:8546');
    const api = await ApiPromise.create({ provider: wsProvider });

    // Simple transaction
    const keyring = new Keyring({type: 'sr25519' });
    const aliceKey = keyring.addFromUri('//Alice',  { name: 'Alice default' });
    console.log(`${aliceKey.meta.name}: has address ${aliceKey.address} with publicKey [${aliceKey.publicKey}]`);

    const ADDR_Bob = '0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22';

    const transfer = await api.tx.balances.transfer(ADDR_Bob, 12345)
        .signAndSend(aliceKey, {era: 0, blockHash: '0x64597c55a052d484d9ff357266be326f62573bb4fbdbb3cd49f219396fcebf78', blockNumber:0,  genesisHash: '0x64597c55a052d484d9ff357266be326f62573bb4fbdbb3cd49f219396fcebf78', nonce: 1, tip: 0, transactionVersion: 1});

    console.log(`hxHash ${transfer}`);

}

main().catch(console.error);
