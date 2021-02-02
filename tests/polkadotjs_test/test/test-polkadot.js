const {describe} = require('mocha');
const {expect} = require('chai');
const exec = require('child_process').exec;
const { ApiPromise, WsProvider } = require('@polkadot/api');

describe('Testing polkadot calls:', function () {
    let api;
    before (async function () {
        const wsProvider = new WsProvider('ws://127.0.0.1:8546');
        api = await ApiPromise.create({ provider: wsProvider });
    });

    after(function () {
       console.log('Stop gossamer');
    });

    it('call api.genesisHash', async function () {
       let genesisHash = await api.genesisHash;
       console.log(`genesis hash: ${genesisHash}`);
       expect(genesisHash).to.be.not.null;
   });
});