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

    const bobKey = keyring.addFromUri('//Bob', {name: 'Bob default'});
    console.log(`${bobKey.meta.name}: has address ${bobKey.address} with publicKey [${bobKey.publicKey}], ${toHexString(bobKey.publicKey)}`);

    const ADDR_Bob = '0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22';
    // bob 5FHneW46xGXgs5mUiveU4sbTyGBzmstUspZC92UhjJM694ty

     // const transfer = await api.tx.balances.transfer(bobKey.address, 12345)
     //     .signAndSend(aliceKey, {era: 0, blockHash: '0x64597c55a052d484d9ff357266be326f62573bb4fbdbb3cd49f219396fcebf78', blockNumber:0,  genesisHash: '0x64597c55a052d484d9ff357266be326f62573bb4fbdbb3cd49f219396fcebf78', nonce: 1, tip: 0, transactionVersion: 1});

    const transfer = await api.tx.balances.transfer(bobKey.address, 12345)
        .signAndSend(aliceKey);

    // console.log(`hxHash ${transfer}`);

    // Make a transfer from Alice to BOB, waiting for inclusion
  // .signAndSend(aliceKey, {era: 0, blockHash: '0x64597c55a052d484d9ff357266be326f62573bb4fbdbb3cd49f219396fcebf78', blockNumber:0,  genesisHash: '0x64597c55a052d484d9ff357266be326f62573bb4fbdbb3cd49f219396fcebf78', nonce: 1, tip: 0, transactionVersion: 1},  ({ events = [], status }) => {
  //   const unsub = await api.tx.balances
  //       .transfer(bobKey.address, 12345)
  //       .signAndSend(aliceKey,   ({ events = [], status }) => {
  //           console.log(`Current status is ${status.type}`);
  //
  //           if (status.isFinalized) {
  //               console.log(`Transaction included at blockHash ${status.asFinalized}`);
  //
  //               // Loop through Vec<EventRecord> to display all events
  //               events.forEach(({ phase, event: { data, method, section } }) => {
  //                   console.log(`\t' ${phase}: ${section}.${method}:: ${data}`);
  //               });
  //
  //               unsub();
  //           }
  //       });

}

main().catch(console.error);

function toHexString(byteArray) {
    return Array.from(byteArray, function(byte) {
        return ('0' + (byte & 0xFF).toString(16)).slice(-2);
    }).join('')
}
