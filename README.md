 # gossamer
 
 ## Golang Polkadot Runtime Environment Implementation  

[![GoDoc](https://godoc.org/github.com/ChainSafeSystems/gossamer?status.svg)](https://godoc.org/github.com/ChainSafeSystems/gossamer)
[![Go Report Card](https://goreportcard.com/badge/github.com/ChainSafeSystems/gossamer)](https://goreportcard.com/report/github.com/ChainSafeSystems/gossamer)
[![CircleCI](https://circleci.com/gh/ChainSafeSystems/gossamer.svg?style=svg)](https://circleci.com/gh/ChainSafeSystems/gossamer)
[![Maintainability](https://api.codeclimate.com/v1/badges/933c7bb58eee9aba85eb/maintainability)](https://codeclimate.com/github/ChainSafeSystems/gossamer/badges)
[![Test Coverage](https://api.codeclimate.com/v1/badges/933c7bb58eee9aba85eb/test_coverage)](https://codeclimate.com/github/ChainSafeSystems/gossamer/test_coverage)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![](https://img.shields.io/twitter/follow/espadrine.svg?label=Follow&style=social)](https://twitter.com/chainsafeth)


## Install

```
go get -v -t -d ./...
```

## Test
```
go test -v ./...
```

#### Note on memory intensive tests
Unfortunately, the free tier for CI's have a memory cap and some tests will cause the CI to experience an out of memory error.
In order to mitigate this we have introduced the concept of **short tests**. If your PR causes an out of memory error please seperate the tests into two groups
like below and make sure to label it `large`:

```
var stringTest = []string {
    "This causes no leaks"
}

var largeStringTest = []string {
    "Whoa this test is so big it causes an out of memory issue"
}

func TestStringTest(t *testing.T) {
    ...
}

func TestLargeStringTest(t *testing.T) {
   	if testing.Short() {
  		t.Skip("\033[33mSkipping memory intesive test for <TEST NAME> in short mode\033[0m")
    } else {
        ...
    }
}
```

## Contributing
- Check out our contribution guidelines: [CONTRIBUTING.md](CONTRIBUTING.md)  
- Have questions? Say hi on [Gitter](https://gitter.im/chainsafe/gossamer)!

## License
_GNU General Public License v3.0_
