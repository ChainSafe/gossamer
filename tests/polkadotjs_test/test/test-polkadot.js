const {describe} = require('mocha');
const {expect} = require('chai');
const { ApiPromise, WsProvider } = require('@polkadot/api');
const sleep = (delay) => new Promise((resolve) => setTimeout(resolve, delay))

describe('Testing polkadot.js/api calls:', function () {
    let api;
    let done = false;

    before (async function () {
        const wsProvider = new WsProvider('ws://127.0.0.1:8546');
        ApiPromise.create({provider: wsProvider}).then( async (a) => {
            api = a;

            do {
                await sleep(5000);
            } while (!done)
        }).finally( () => process.exit());
    });


    beforeEach ( async function () {
        // this is for handling connection issues to api, if not connected
        //  wait then try again, if still not corrected, close test
        this.timeout(5000);

        if (api == undefined) {
            await sleep(2000);
        }

        if (api == undefined) {
            process.exit(1);
        }
    });

     after(function () {
         done = true;
     });

    describe('chain queries', () => {
        it('call api.genesisHash', async function () {
            const genesisHash = await api.genesisHash;
            expect(genesisHash).to.be.not.null;
            expect(genesisHash).to.have.lengthOf(32);
        });

        it('call api.runtimeMetadata', async function () {
            const runtimeMetadata = await api.runtimeMetadata;
            expect(runtimeMetadata).to.be.not.null;
            expect(runtimeMetadata).to.have.deep.property('magicNumber')
        });
    });

});