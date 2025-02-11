# Contribution Guidelines

Thank you for your interest in our implementation of the Polkadot Runtime Environment Implementation! We're excited to get to know you and work with you on gossamer. We've put together these guidelines to help you figure out how you can help us.

At any point in this process feel free to reach out on [Discord](https://discord.gg/M5XgXGRv) with any questions or to say Hello :)

## Getting Started

Generally, it is important to have a basic understanding of Polkadot and the Polkadot Runtime Environment. Having a stronger understanding will allow you to make more significant contributions. We've put together a list of resources that can help you develop this fundamental knowledge.

The Web3 Foundation has a [Polkadot Wiki](https://wiki.polkadot.network/docs/getting-started) that would help both part-time and core contributors to the project in order to get up to speed. Our [Gossamer docs](https://chainsafe.github.io/gossamer/) also has some helpful resources.

The [Polkadot Runtime Specification](https://research.web3.foundation/en/latest/_static/pdfview/viewer.html?file=../pdf/polkadot_re_spec.pdf) serves as our primary specification, however it is currently in its draft status so things may be subject to change.

To understand more about Parachain Protocol please refer to [The Polkadot Parachain Host Implementers Guide](https://paritytech.github.io/polkadot-sdk/book/index.html).
Additionally information about design that is done by our team for some modules refer to our [design docs](https://github.com/ChainSafe/gossamer/tree/development/docs/docs/design)
or [Gossamer blogposts](https://blog.chainsafe.io/gossamer/).

And there are many more articles and videos that we are willing to share if you are interested. So please come by to our [Discord](https://discord.gg/M5XgXGRv) channel to ask for help or simply say hi.

For coding style, you may refer to the [code style](CODE_STYLE.md) document which we keep up to date with coding style conventions we have for this repository.

## Contribution Steps
1. Make sure you're familiar with our contribution guidelines (this document)!
2. Find an issue you want to work on. We have a ["good first issue" label on github](https://github.com/ChainSafe/gossamer/issues?q=is%3Aissue%20state%3Aopen%20label%3A"good%20first%20issue). To avoid duplicate efforts or working on outdated tasks, please clarify your intentions in the issue comments before starting. This step ensures there are no parallel executions of the same issue and confirms its validity for implementation.
3. Create your own fork of this repository.
4. Make your changes in your local fork.
5. If you've made a code change, make sure to lint and test your changes.
   Changes that only affect a single file can be tested with

    ```sh
    go test <file_you_are_working_on>
    ```

   Sometimes you may need to create mocks for interfaces, in that case, add a go generate comment. For example, for interface `Client` in the `dot/telemetry` package, the comment would be:

    ```go
    //go:generate mockgen -destination=mock_myinterface_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client
    ```

   This will generate a Go file `mock_myinterface_test.go` with the `Client` mock. Note this is only accessible
   in your current package since it's written to a `_test.go` file. We prefer to generate mocks locally where they are needed instead of sharing them to reduce package dependency and the Go API 'noise'.

   Before opening a pull request be sure to run the linter

    ```sh
    make lint
    ```
6. **Add licenses to new Go and Proto files**
    If you added any new file, run `make license` to setup all licenses on relevant files.
    If you do not have `make` available, you can copy paste the command from the Makefile's `license:` block and run that instead.
7. Make an open pull request when you're ready for it to be reviewed. We review PRs on a regular basis. See Pull request etiquette for more information.
8. You may be asked to sign a Contributor License Agreement (CLA). We make it relatively painless with CLA-bot.

## Note on memory intensive tests

Unfortunately, the free tier for CI's have a memory cap and some tests will cause the CI to experience an out of memory error.
In order to mitigate this we have introduced the concept of **short tests**. If your PR causes an out of memory error please separate the tests into two groups
like below and make sure to label it `large`:

```go
var stringTest = []string {
    "This causes no leaks"
}

var largeStringTest = []string {
    "Whoa this test is so big it causes an out of memory issue"
}

func TestStringTest(t *testing.T) {
    // ...
}

func TestLargeStringTest(t *testing.T) {
    if testing.Short() {
        t.Skip("\033[33mSkipping memory intesive test for <TEST NAME> in short mode\033[0m")
    }
    // ...
}
```


## PR & Commit Standards
The purpose of this section is to outline the standard naming conventions approved by the Gossamer team for Pull Requests and commit messages. The reasoning is to have improved understanding and auto generated changelogs for releases.

### PR and final commit message should follow:

`**TYPE:[SCOPE]: brief description**`

**TYPEs**:

- **fix** = patches a bug or a resolves a particular issue
- **feat** = introduces new functionality
- **chore** = regular code maintenance
- **docs** = anything related to documentation
- **style** = updates related to styling (e.g. logs)
- **test** = anything related to tests
- **refactor** = refactoring a section of the code base

**[SCOPE]**

- name of primary package that changes were introduced (e.g. lib/runtime)

### Example PR names:

- fix(dot/rpc): fixed return type for chain head

- feat(dot/rpc): Add support for submit and watch extrinisic

- docs: Updated usage section for running a gossamer node

- tests(lib/runtime): Added additional test coverage for allocator

- chore(lib/runtime): Bumped wasmer to 1.0

- style(cmd/gossamer): Updated CLI arguments format

- refactor(lib/trie): Refactored how trie is pruned


> For improved Git commit messages refer to:  
> https://www.freecodecamp.org/news/writing-good-commit-messages-a-practical-guide/


## Merge Process

### In General

A Pull Request (PR) needs to be reviewed and approved by project maintainers.
If a change does not alter any logic (e.g. comments, dependencies, docs), then it may be tagged
`C-simple` and merged faster.

###  Labels

The set of labels and their description can be found [labels.yml](/.github/labels.yml).
To change update this file and CI will automatically add/remove changed labels.

### Process

1. Please use our [Pull Request Template](./PULL_REQUEST_TEMPLATE.md) and make sure all relevant
   information is reflected in your PR.
2. Please tag each PR with minimum one `S-*` (scope) label. The respective `S-*` labels should signal the
   component that was changed, they are also used by downstream users to track changes and to
   include these changes properly into their own releases.
3. If you’re still working on your PR, please submit as “Draft”. Once a PR is ready for review change
   the status to “Open”, so that the maintainers get to review your PR. Generally PRs should sit for
   48 hours in order to garner feedback. It may be merged before if all relevant parties had a look at it.
4. PRs will be able to be merged once all reviewers' comments are addressed and CI is successful.

**Noting breaking changes:**
When breaking APIs, the PR description should mention what was changed alongside some examples on how
to change the code to make it work/compile.

## Contributor Responsibilities

We consider two types of contributions to our repo and categorize them as follows:

### Part-Time Contributors

Anyone can become a part-time contributor and help out on gossamer. Contributions can be made in the following ways:

- Engaging in Discord conversations, asking questions on how to contribute to the project
- Opening up Github issues to contribute ideas on how the code can be improved
- Opening up PRs referencing any open issue in the repo. PRs should include:
    - Detailed context of what would be required for merge
    - Tests that are consistent with how other tests are written in our implementation
- Proper labels, milestones, and projects (see other closed PRs for reference)
- Follow up on open PRs
    - Have an estimated timeframe to completion and let the core contributors know if a PR will take longer than expected

### Core Contributors
Core contributors are currently comprised of members of the ChainSafe Systems team.

### Join Core team
If you have an intention of joining the core team, please 