---
layout: default
title: Integration Tests
permalink: /testing-and-debugging/testing
---

## Gossamer Test Suite

To run Gossamer unit tests run the following command:

```
make test
``` 

The above command will run all tests on project files with a timeout set for 20 minutes, and generate a coverage report in root `c.out`. 

You can view the coverage report through HTML by running the below command after running the above unit tests, or by visiting our [code coverage report here](https://app.codecov.io/gh/ChainSafe/gossamer).

```
go tool cover -html=c.out -o cover.html
```

Proceed to open `cover.html` in your preferred browser. 

### Gossamer Integration Tests

Running Gossamer's integration tests with the below commands will build a Gossamer binary, install required dependencies, and then proceeds to run the provided set of tests. Integration tests can also be run within a docker container.
 

To run Gossamer integration tests in **stable** mode run the following command:

```
make it-stable
```

To run Gossamer integration tests in **stress** mode run the following command:

```
make it-stress
```

To run Gossamer integration tests against **GRANDPA** in stress mode run the following command:

```
make it-grandpa
```

To run Gossamer **RPC** integration tests run the following command:

```
make it-rpc
```

To run Gossamer **Sync** integration tests run the following command:

```
make it-sync
```

To run Gossamer **Polkadot.js** integration tests run the following command:

```
make it-polkadotjs
```

