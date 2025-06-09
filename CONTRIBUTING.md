## Setting up the environment

This repository uses [`hermit`](https://cashapp.github.io/hermit/).
Other package managers may work but are not officially supported for development.

## Running locally

While developing a new feature or fix, it may be helpful to use a local copy to test changes.

```sh
$ go run cmd/kustomize-lint/main.go --debug lint path/to/my/kustomization
```

## Running tests

```sh
$ go test ./...
```

## Linting and formatting

This repository uses [golangci-lint](https://golangci-lint.run/) to lint and format the code in the repository.

To lint:

```sh
$ golangci-lint run
```

To format:

```sh
$ golangci-lint fmt
```

## Publishing and releases

Changes made to this repository via the automated release PR pipeline will publish to a GitHub release.
