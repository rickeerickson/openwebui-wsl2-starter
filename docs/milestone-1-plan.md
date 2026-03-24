# Milestone 1: Foundation

Status: **Complete**

## Goal

Go module, CI pipeline, and three shared `internal/` packages
(logging, exec, config) with full test coverage and mutation
testing. No CLI subcommands yet; those are Milestone 2.

## Prerequisites

- [x] Go installed (1.26.1)
- [x] golangci-lint installed (2.11.4)
- [x] gremlins installed (0.6.0)

## Wave 1: Project Scaffold (sequential)

| PR  | Scope                                   | Status    |
|-----|-----------------------------------------|-----------|
| 1.1 | go.mod, project structure, Makefile     | committed |

## Wave 2: Independent Packages (3 parallel agents)

| PR  | Scope                                   | Status    |
|-----|-----------------------------------------|-----------|
| 1.3 | `internal/logging` with tests           | committed |
| 1.5 | `internal/exec` retry with tests        | committed |
| 1.6 | `internal/config` struct, YAML, validate| committed |

## Wave 3: Dependent Packages (2 parallel agents)

| PR  | Scope                                   | Status    |
|-----|-----------------------------------------|-----------|
| 1.4 | `internal/exec` runner + allowlist      | committed |
| 1.7 | `internal/config` env + flag override   | committed |

## Wave 4: Infrastructure (2 parallel agents)

| PR  | Scope                                   | Status    |
|-----|-----------------------------------------|-----------|
| 1.2 | CI workflow (GitHub Actions)            | committed |
| 1.8 | `.golangci.yml` with security linters   | committed |

## Wave 5: Mutation Testing (sequential)

| PR  | Scope                                   | Status    |
|-----|-----------------------------------------|-----------|
| 1.9 | Mutation test, fix gaps, merge          | committed |

## Dependency Graph

```text
1.1 --+-- 1.3 (logging) --+-- 1.4 (runner)
      |-- 1.5 (retry) ----+
      |-- 1.6 (config) ------- 1.7 (override)
      |-- 1.2 (CI)
      +-- 1.8 (golangci)
                                 +-- 1.9
```

## Agent File Boundaries

| Agent | Files Owned                             |
|-------|-----------------------------------------|
| Main  | go.mod, cmd/ow/, Makefile, .git*        |
| A     | internal/logging/*                      |
| B     | internal/exec/retry.go, retry_test.go   |
| C     | internal/config/config.go, testdata/*   |
| D     | internal/exec/runner.go, allowlist.go   |
| E     | internal/config/resolve.go              |
| F     | .github/workflows/go.yml                |
| G     | .golangci.yml                           |

## Verification Criteria

- `make test` passes with `-race`
- `make lint` passes
- `make build-all` cross-compiles both platforms
- Mutation score >= 80% on `internal/` packages
- `git push` triggers green CI
