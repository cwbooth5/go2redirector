# Contributing

Any help is appreciated. Here are some tips for development and contributing to this project.

## Filing Bugs
If there are any issues with any go2redirector releases, create an issue in our github page for the repository.

## Running Tests
Tests should be running and passing before submitting a PR. There is no restriction on coverage percentage, but adding tests for your new code is encouraged.

You can run tests with `go test` in the root directory. For coverage with an HTML report, use this.

```
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Submitting Pull Requests
Ideally, each PR would have either an explanation of _why_ things are being changed, or an associated bug/issue describing the need for the change.