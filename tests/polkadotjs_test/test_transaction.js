// Import
const { ApiPromise, WsProvider } = require('@polkadot/api');
const {Keyring } = require('@polkadot/keyring');

async function main() {
    const wsProvider = new WsProvider('ws://127.0.0.1:8546');
    const api = await ApiPromise.create({ provider: wsProvider });

    // wait for block 1
    const unsub = await api.rpc.chain.subscribeNewHeads(async (lastHeader) => {
        console.log(`latest block: #${lastHeader.number} `);
        if (lastHeader.number > 1) {
            unsub()
        }
    });

    // Simple transaction
    const keyring = new Keyring({type: 'sr25519' });
    const aliceKey = keyring.addFromUri('//Alice',  { name: 'Alice default' });
    console.log(`${aliceKey.meta.name}: has address ${aliceKey.address} with publicKey [${aliceKey.publicKey}]`);

    const bobKey = keyring.addFromUri('//Bob', {name: 'Bob default'});
    console.log(`${bobKey.meta.name}: has address ${bobKey.address} with publicKey [${bobKey.publicKey}], ${toHexString(bobKey.publicKey)}`);

    const transfer = await api.tx.balances.transfer(bobKey.address, 12345).signAndSend(aliceKey);
    console.log(`transaction hash: ${transfer}`);
}

main().catch(console.error);

function toHexString(byteArray) {
    return Array.from(byteArray, function(byte) {
        return ('0' + (byte & 0xFF).toString(16)).slice(-2);
    }).join('')
}
