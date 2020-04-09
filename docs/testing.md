# Testing

This project is following the patter of TDD (Test-Driven-Development) with BDD (Behavioral-Driven-Development). This means tests are playing a primary role and we take them seriously. It is expected from PRs to add, modify or delete tests on case by case scenario. To contribute to need to be familiar with:

* [Ginkgo](https://github.com/onsi/ginkgo) - BDD Testing Framework for Go
* [GoMock](https://github.com/golang/mock) - a mocking framework for Go

## GoMock

Once you have installed Go, install the mockgen tool.

To get the latest released version use:

```zsh
GO111MODULE=on go get github.com/golang/mock/mockgen@latest
```

## Ginkgo

```zsh
go get -u github.com/onsi/ginkgo/ginkgo  # installs the ginkgo CLI
go get -u github.com/onsi/gomega/...     # fetches the matcher library
```

## How to test

When we are writing unit tests, we want to make sure that our **unit** works as expected. Simple put, for a given input it returns an expected output. Most of the tutorials are showing very simple examples (for a good reason) where there is a function that receives two integers and returns the sum of them. However some functions under test relies on another function which you have no control. That usually happens when you are using external services, like communicating with an API or a web service or many other things. Notice that the GCP Operator interacts with both the Kubernetes API Server and the Google API Server. The important point is that your test can't control what that dependency returns back to your function or how it behaves. There are many things can go wrong here, the network, the external service system, and many other things. So, I will use *mocking* techniques to **simulate** both an expected and an unexpected behavior of this external dependency.

A **stub** is a controllable replacement for an existing dependency in the system. In programming, you use **stubs** to get around around the problem of external dependencies. By using a system, you can test your code without dealing with the dependency directly.

So you can't test something? Add a layer that wraps the calls to that something, and then mimic that layer in your tests. To do that, Go makes heavily use of interfaces which is a type with a set of method signatures. If _any type_ implements those methods, it _satisfies_ the interface and be recognized by the interface's type. The trick is to use `GoMock` to create mocks for interfaces that include functions that simulate the expected (or not) behavior from the external dependency.


To make the code easier to read, we use `Ginkgo` which is the dominant testing framework in Kubernetes friendly ecosystems, like this one.

For example if you make any modification to the file `./pkg/gcpclient/client.go` then you need to update the mocks for that. The `mockgen` binary helps us by doing this task for us, by running `mockgen -destination=../util/mocks/$GOPACKAGE/client.go -package=$GOPACKAGE -source client.go`.

The same stands for the other packages, such as `./pkg/controller/projectclaim/projectclaim_controller.go` where we need to run `mockgen -destination=../../util/mocks/$GOPACKAGE/customeresourceadapter.go -package=$GOPACKAGE github.com/openshift/gcp-project-operator/pkg/controller/projectclaim CustomResourceAdapter`.

This is if you want to test quickly some changes, otherwise you don't need any of those commands to be run manually, as a simple `make` should suffice and run them behind-the-scenes for you.