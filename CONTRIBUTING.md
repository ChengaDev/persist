# Contributing to psst

Thank you for your interest in contributing!

## Setting up

```sh
git clone https://github.com/ChengaDev/persist.git
cd persist
go mod download
make build
```

**Requirements:** Go 1.19+, `make`

## Running tests

```sh
make test-short   # fast — skips slow Argon2id tests
make test         # full suite (takes ~30s due to Argon2id)
```

## Making changes

1. Fork the repository and create a branch from `main`
2. Make your changes
3. Add or update tests to cover the change
4. Run `make test` and ensure everything passes
5. Open a pull request

## Test coverage

All new code must be accompanied by tests. This includes:

- New internal packages — add a `*_test.go` file alongside the package
- New CLI commands — add tests to `cmd/cmd_test.go` covering at least the happy path and the main error cases
- Bug fixes — add a test that would have caught the bug

The root `main.go` is the only file exempt from this requirement, as it only calls `cmd.Execute()`.

Run the full suite before opening a PR:

```sh
make test
```

## Pull request guidelines

- Keep PRs focused — one concern per PR
- Cryptography changes (`internal/crypto/`) must include a clear explanation of
  why the change is safe
- Do not change Argon2id parameters without discussion — this affects all existing
  encrypted entries
- All new commands must follow the zero-leakage policy (no secrets as flag values)

## Reporting a security vulnerability

See [SECURITY.md](SECURITY.md). Do not open a public issue.
