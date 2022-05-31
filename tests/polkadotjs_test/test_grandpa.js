const { ApiPromise, WsProvider } = require('@polkadot/api');
const {Keyring } = require('@polkadot/keyring');

async function main() {
    // Construct

    const wsProvider = new WsProvider('ws://127.0.0.1:8546');
    // const wsProvider = new WsProvider('ws://127.0.0.1:9944');
    const api = await ApiPromise.create({ provider: wsProvider });

    // grandpa_proveFinality
    const proveBlockNumber = 10;
    const proveFinality = await api.rpc.grandpa.proveFinality(proveBlockNumber);
    console.log(`proveFinality: ${proveFinality}`);
}

main().catch(console.error);