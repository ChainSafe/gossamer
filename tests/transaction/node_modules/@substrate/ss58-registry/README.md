# SS58 Registry

[![GitHub license](https://img.shields.io/badge/license-Apache2-green)](#LICENSE) [![GitLab Status](https://gitlab.parity.io/parity/ss58-registry/badges/main/pipeline.svg)](https://gitlab.parity.io/parity/ss58-registry/pipelines)

A list of known [SS58](https://docs.substrate.io/v3/advanced/ss58/) account types as an enum, typically used by the Polkadot, Kusama or Substrate ecosystems.

These are derived from the [json data file](ss58-registry.json) in this repository which contains entries like this:

```js
{
	"prefix": 5,                       // unique u16
	"network": "astar",                // unique no spaces
	"displayName": "Astar Network",    //
	"symbols": ["ASTR"],               // symbol for each instance of the Balances pallet (usually one)
	"decimals": [18],                  // decimals for each symbol listed
	"standardAccount": "*25519",       // Sr25519, Ed25519 or secp256k1
	"website": "https://astar.network" // website or code repository of network
},
```

## Process

1. Fork and clone this repo.

2. Add an additional account type to `ss58-registry.json` (contiguous prefixes are better).

3. Bump the minor (middle) version number of the `Cargo.toml` by running:
```
cargo install cargo-bump && cargo bump minor
```
4. Run git stage, commit, push and then raise a pull request.

5. Once the PR has landed, one of the admins can
[create a new release](https://github.com/paritytech/ss58-registry/releases/new).
This will release the new version to [crates.io](https://crates.io/crates/ss58-registry)

## Licensing

Apache-2.0
