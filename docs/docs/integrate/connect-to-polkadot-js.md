---
layout: default
title: Connect to Polkadot.js
permalink: /integrate/connect-to-polkadot-js/
---

# Connecting to Polkadot.js 

To view your running Gossamer node with a UI, <a target="_blank" rel="noopener noreferrer" href="https://github.com/polkadot-js/apps">Polkadot has created a handy app</a>, which you can use here: <a target="_blank" rel="noopener noreferrer" href="https://polkadot.js.org/apps/">https://polkadot.js.org/apps/</a>

If using Polkadot's hosted app, you will need to ensure your node has the `--rpc-external`, `--ws` & `--ws-external` flags, if you are running the app locally, just ensure the rpc & websocket servers are running (`--rpc` && `--ws`)

For example:
```
bin/gossamer --rpc --ws --wsport 8546 --rpcmods system,author,chain,state,account,rpc --key alice
```

### Connecting the app to your node

You'll need to setup the polkadot.js/apps to use a custom endpoint to connect to your gossamer node.  Open [polkadot.js.org/apps](https://polkadot.js.org/apps).

Once you've opened the app in your browser, you should see it connected to the Polkadot network: 

<img src="/assets/tutorial/connect-1.png" />

In the top left hand corner, click the logo to open the network selection modal: 

<img src="/assets/tutorial/connect-2.png" />

Next, at the bottom of this menu is a "Development" dropdown, click to open that

<img src="/assets/tutorial/connect-3.png" />

Now you should see a text area with the label "custom endpoint", here you add your local node's websocket address, usually "ws://127.0.0.1:8586",
click the Save icon on the right of the text box to save the endpoint.

<img src="/assets/tutorial/connect-4.png" />

Finally, click the "Switch" button at the top of this modal:

<img src="/assets/tutorial/connect-5.png" />

Congratulations, you've successfully connected to your Gossamer node!

<img src="/assets/tutorial/connect-6.png" />

