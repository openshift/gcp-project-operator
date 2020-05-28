# Contributing

### How to become a GCP Project contributor

## Code of Conduct

As contributors and maintainers of this GCP Project Operator, we do respect all people who contribute through reporting issues, posting feature requests, updating documentation, submitting pull requests or patches, and other activities.
We are committed to making participation in this project a harassment-free experience for everyone, regardless of level of experience, gender, gender identity and expression, sexual orientation, disability, personal appearance, body size, race, ethnicity, age, religion, or nationality. In short, be excellent to each other.

## Finding issues to work on

* ["good-first-issue](https://github.com/openshift/gcp-project-operator/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) - issues where they are easy to complete even for beginners

* ["help wanted"](https://github.com/openshift/gcp-project-operator/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) - issues where we currently have no resources to work on them as there are other pressing matters

Once you've discovered an issue to work on:

* Add a comment mentioning that you plan to work on this issue and assign it to yourself.
* Send a PR out that mentions the issue in the _commit_ message.

## Contributing A Patch

If you are new to open-source contribution, please read the this [guide](https://developers.redhat.com/articles/command-line-heroes-game-pull-request/) to get yourself familiar with the basics.

## Writing tests

As a best practice, the project requires tests to be submitted at the same PR with the code. If you are developing a new feature, please remember writting tests for it as well. See the relevant [testing documenation](./docs/testing.md)

## An example PR template

Here is a [PR template](./docs/PULL_REQUEST_TEMPLATE.md) which can be used by contributors. 
After having both `/lgtm` and `/approve` labels from approvers and reviewers, your PR will be merged by `Openshift Merge Bot`
To prevent any accidental merge, marking the PR as `work in progress` by adding the `[WIP]` tag would be nice
