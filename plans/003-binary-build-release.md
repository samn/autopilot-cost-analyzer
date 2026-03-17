# Plan 003: Binary Build & Release

## Problem

We need a repeatable process to build release binaries and publish them as
GitHub Releases so users can download a single static binary.

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Build tool | `make` | Universal, zero dependencies, familiar to Go developers |
| Version injection | `ldflags` | Standard Go pattern—no generated files or extra dependencies |
| Binary linking | Static (`CGO_ENABLED=0`) | Single binary, no glibc dependency, portable across distros |
| Strip symbols | `-s -w` in ldflags | Smaller binary, not needed for production |
| Release trigger | Git tag `v*` | Conventional, works with `git describe` and semver |
| Release creation | `softprops/action-gh-release` | Well-maintained, handles asset upload in one step |
| Changelog extraction | `awk` in workflow | No extra tooling; CHANGELOG.md is the single source of truth |
| Initial target | `linux/amd64` | Primary deployment target (GKE nodes / CI runners) |
| Cross-compilation | Supported via `GOOS`/`GOARCH` Make vars | Easy to add more targets later |

## Files

| File | Purpose |
|---|---|
| `cmd/version.go` | `version` subcommand; holds `ldflags` variables |
| `cmd/version_test.go` | Tests for the version command |
| `Makefile` | Build, test, lint, clean targets |
| `.github/workflows/release.yaml` | CI workflow: test → build → release on tag push |
| `.gitignore` | Updated to ignore `dist/` |
| `CHANGELOG.md` | Updated with build/release entries |

## How to Release

1. Update `CHANGELOG.md`: move items from `[Unreleased]` into a new
   `[X.Y.Z] - YYYY-MM-DD` section.
2. Commit the changelog update.
3. Tag the commit:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
4. The `release.yaml` workflow runs automatically:
   - Runs the full test suite.
   - Builds a static `linux/amd64` binary with version metadata baked in.
   - Creates a GitHub Release with the changelog section as the body and the
     binary attached.

## How to Build Locally

```bash
# Default: linux/amd64
make build

# macOS arm64
make build GOOS=darwin GOARCH=arm64

# Check version info
./dist/gke-cost-analyzer-linux-amd64 version
```

## Adding More Platforms

Add a matrix to the release workflow or add more `make build` invocations:

```yaml
strategy:
  matrix:
    include:
      - goos: linux
        goarch: amd64
      - goos: darwin
        goarch: arm64
```

Each combination produces `dist/gke-cost-analyzer-<os>-<arch>`.
