const {describe} = require('mocha');
const {expect} = require('chai');
const exec = require('child_process').exec;
const { ApiPromise, WsProvider } = require('@polkadot/api');

describe('Testing polkadot calls:', function () {
    let api;
    before (function () {
         exec('pwd', function callback(error, stdout, stderr) {
             console.log('out ' + stdout)
         });
        const wsProvider = new WsProvider('ws://127.0.0.1:8546');
        api = ApiPromise.create({ provider: wsProvider });
    });

    after(function () {
       console.log('Stop gossamer');
    });

    it('call api.genesisHash', function () {
       let genesisHash = api.genesisHash;
       // console.log(`genesis hash: ${genesisHash}`);
        expect(genesisHash).to.not.be.null;
   });
});