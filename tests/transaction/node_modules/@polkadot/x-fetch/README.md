# @polkadot/x-fetch

A cross-environment fetch.

Install it via `yarn add @polkadot/x-fetch`

```js
import { fetch } from '@polkadot/x-fetch';

...
const response = await fetch('https://example.com/something.json');
const json = await response.json();
```
