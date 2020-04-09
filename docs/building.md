# Building the GCP Project Operator

## Prerequisites

* A recent Go distribution (>=1.14) with enabled Go modules

* Docker or Podman

* The `make` binary

## Downloading the source

```zsh
# inside your $GOPATH/src/*
$ git clone https://github.com/openshift/gcp-project-operator
$ cd gcp-project-operator
```

## Compiling gcp-project-operator

```zsh
$ make
```

This will run the tests and build the binaries which they can be found at `gcp-project-operator/build/_output/bin`