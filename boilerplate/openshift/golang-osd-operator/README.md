# Conventions for OSD operators written in Go

This convention is suitable for both cluster- and hive-deployed operators.

The following components are included:

## `make` targets and functions.
**Note:** Your repository's main `Makefile` needs to be edited to include the
"nexus makefile include":

```
include boilerplate/generated-includes.mk
```

## Code coverage
- A `codecov.sh` script, referenced by the `coverage` `make` target, to
run code coverage analysis per [this SOP](https://github.com/openshift/ops-sop/blob/ff297220d1a6ac5d3199d242a1b55f0d4c433b87/services/codecov.md).

- A `.codecov.yml` configuration file for
  [codecov.io](https://docs.codecov.io/docs/codecov-yaml). Note that
  this is copied into the repository root, because that's
  [where codecov.io expects it](https://docs.codecov.io/docs/codecov-yaml#can-i-name-the-file-codecovyml).

## Linting and other static analysis with `golangci-lint`

- A `gocheck` `make` target, which
- ensures the proper version of `golangci-lint` is installed, and
- runs it against
- a `golangci.yml` config.

## More coming soon
