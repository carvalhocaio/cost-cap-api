# go-template

A minimalist, modern Go project template preconfigured with:
- **Go 1.27+** with a module-based layout (`cmd/` + `internal/`)
- **[golangci-lint v2](https://golangci-lint.run/)** for linting and formatting (`gofumpt` + `goimports`)
- **[govulncheck](https://go.dev/doc/security/vuln/)** for dependency vulnerability scanning, pinned as a `go tool`
- **[Pre-commit](https://pre-commit.com/)** git hooks for code hygiene and security
- **`go test`** with the race detector and a table-driven smoke test
- **Idiomatic Makefile** for development workflow automation
- **GitHub Actions CI** matching local checks

---

## 🚀 Quickstart

### 1. Prerequisites

- [Go](https://go.dev/dl/) 1.27+
- [golangci-lint](https://golangci-lint.run/docs/welcome/install/) v2
- [pre-commit](https://pre-commit.com/#install) (e.g. `uv tool install pre-commit`)

### 2. Using this template

Click **"Use this template"** on GitHub or clone the repository:

```bash
git clone https://github.com/<username>/<repo-name>.git
cd <repo-name>
```

### 3. Rename the project

Run the renaming helper to set your module path. The binary name is taken from the last path segment:

```bash
make rename MODULE=github.com/<username>/my-cli
```

This renames `cmd/app_name/` and updates `go.mod`, imports, the `Makefile`, `.golangci.yml` and this README.

### 4. Install dependencies and git hooks

```bash
make deps
make hooks
```

### 5. Build and run

```bash
make build
./bin/app_name -name Gopher
./bin/app_name -version
```

---

## 🛠️ Available Commands

| Command | Description |
|---|---|
| `make help` | Show all available commands |
| `make deps` | Download module dependencies |
| `make tidy` | Add missing and remove unused dependencies |
| `make hooks` | Install pre-commit hooks into `.git/hooks` |
| `make hooks-run` | Run pre-commit checks on all files |
| `make build` | Build the binary into `bin/` (version injected from `git describe`) |
| `make run ARGS="..."` | Run the application with `go run` |
| `make test` | Run tests with the race detector |
| `make cover` | Run tests and print a coverage report |
| `make lint` | Check code with `golangci-lint` |
| `make lint-fix` | Automatically fix linting issues |
| `make format` | Format code with `golangci-lint fmt` |
| `make format-check` | Check code formatting without modifying |
| `make audit` | Audit dependencies for vulnerabilities with `govulncheck` |
| `make ci` | Run full verification pipeline locally (`lint`, `format-check`, `audit`, `test`) |
| `make clean` | Remove build artifacts and test cache |
| `make rename MODULE=...` | Rename module and binary |

---

## 📁 Project Structure

```text
.
├── .github/workflows/ci.yml   # GitHub Actions CI workflow
├── cmd/
│   └── app_name/              # CLI entrypoint (renamed via make rename)
│       └── main.go
├── internal/
│   └── app/                   # Application logic
│       ├── app.go
│       └── app_test.go        # Initial smoke test
├── scripts/
│   └── rename/                # Project rename helper
│       └── main.go
├── .gitattributes
├── .gitignore
├── .golangci.yml
├── .pre-commit-config.yaml
├── go.mod
├── Makefile
└── README.md
```

---

## 🔄 Continuous Integration

The workflow in `.github/workflows/ci.yml` runs on every push and pull request to `main`/`master`:

1. `go mod tidy -diff` — ensures `go.mod`/`go.sum` are tidy
2. `golangci-lint run` — lint
3. `golangci-lint fmt --diff` — format check
4. `go tool govulncheck ./...` — vulnerability audit
5. `go test -race ./...` — tests
6. `go build ./...` — build

Run the same checks locally with `make ci`.
