# Testing

Tests are playing a primary role and we take them seriously.
It is expected from PRs to add, modify or delete tests on case by case scenario.
To contribute you need to be familiar with:

* [Ginkgo](https://github.com/onsi/ginkgo) - BDD Testing Framework for Go
* [GoMock](https://github.com/golang/mock) - a mocking framework for Go

## GoMock

To get the latest released version of `mockgen` tool use:

```zsh
GO111MODULE=on go get github.com/golang/mock/mockgen@latest
```

## Ginkgo

```zsh
go get -u github.com/onsi/ginkgo/ginkgo  # installs the ginkgo CLI
go get -u github.com/onsi/gomega/...     # fetches the matcher library
```

## How to run the tests

* You can run the tests using `make gotest` or `go test ./...`
* You can generate the mocks using `make generate`
* You can generate the new testing coverage badge using `make coverage` (created the badge and `coverage.out` report)

To see more details about the testing coverage, use `go tool cover -html=coverage.out` in your local machine.

## How to test

When we are writing unit tests, we want to make sure that our **unit** works as expected.
Simple put, for a given input it returns an expected output.
However some functions rely on another function which you have no control.
That usually happens when you are using external services, like communicating with an API or a web service or many other things.
Notice that the GCP Operator interacts with both the Kubernetes API Server and the Google API Server.
There are many things can go wrong here, the network, the external service system, and many other things.
So, we recommend you to use *mocking* techniques to **simulate** both an expected and an unexpected behavior of this external dependency.
To do that, we use _stubs_.
A **stub** is a controllable replacement for an existing dependency in the system.
In programming, you use **stubs** to get around around the problem of external dependencies.

So you can't test something directly?
Add a layer that wraps the calls to that something, and then _mimic that layer_ in your tests.
To do that, Go makes heavily use of interfaces which is a type with a set of method signatures.
If _any type_ implements those methods, it _satisfies_ the interface and gets recognized by the interface's type.

To make _mocking_ easier we are using `GoMock` to create mocks for interfaces that include functions that simulate the expected (or not) behavior from the external dependency.

To make the code easier to read, we use `Ginkgo` which is the dominant testing framework in Kubernetes friendly ecosystems.

### Example

If you want to test quickly some changes, you can run directly the `mockgen` binary.
Otherwise a simple `make` should suffice and run them behind-the-scenes for you.

For instance, if you make any modification to the file `./pkg/gcpclient/client.go` then you need to update its mocks.

The `mockgen` binary helps us by doing this task for us, by running `mockgen -destination=../util/mocks/$GOPACKAGE/client.go -package=$GOPACKAGE -source client.go`.

The same stands for the other packages, such as `./pkg/controller/projectclaim/projectclaim_controller.go` where we need to run `mockgen -destination=../../util/mocks/$GOPACKAGE/customeresourceadapter.go -package=$GOPACKAGE github.com/openshift/gcp-project-operator/pkg/controller/projectclaim CustomResourceAdapter`.

