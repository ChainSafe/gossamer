# @polkadot/x-global

A cross-environment global object. checks for global > self > window > this.

Install it via `yarn add @polkadot/x-global`

```js
import { xglobal } from '@polkadot/x-global';

console.log(typeof xglobal.TextEncoder);
```
