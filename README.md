# common-go

This is where I keep shared Go code until I decide to split it into separate repositories. It includes database helpers, caching, queues, rate limiting, Temporal utilities and other pieces reused across my projects. Packages move out when they need their own release cycle.

For new code, use [`ctxerrors/commerr`](https://github.com/psyb0t/ctxerrors) instead of `errors`, [`aichteeteapee`](https://github.com/psyb0t/aichteeteapee) instead of `http`, and [`slogging/handlers/loki`](https://github.com/psyb0t/slogging) instead of `slogging/loki`. The older packages still ship. `errors` re-exports the same sentinel values, preserving `errors.Is` comparisons.

[![Go Reference](https://pkg.go.dev/badge/github.com/psyb0t/common-go.svg)](https://pkg.go.dev/github.com/psyb0t/common-go)
[![CI](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml)
[![coverage](https://raw.githubusercontent.com/psyb0t/common-go/badges/coverage.svg)](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml)
[![version](https://raw.githubusercontent.com/psyb0t/common-go/badges/version.svg)](https://github.com/psyb0t/common-go/releases)
[![license](https://raw.githubusercontent.com/psyb0t/common-go/badges/license.svg)](LICENSE)
[![imported by](https://raw.githubusercontent.com/psyb0t/common-go/badges/importers.svg)](https://github.com/psyb0t/common-go/blob/badges/importers.md)

> Log scope propagation moved from `common-go/scope` to [`ctxscope`](https://github.com/psyb0t/ctxscope) in v0.4.0. The core API is the same under its new package name (`scope.Set` becomes `ctxscope.Set`), with `NewHandler` for automatic context-scope propagation. Pin `v0.3.x` to keep the old import path.

## SQLite with GORM

[`db/sqlite/gorm`](db/sqlite/gorm) provides a pure-Go modernc SQLite dialector. Use `sqlitegorm.Open(dsn)` with `gorm.Open` and set `gorm.Config.TranslateError` to translate primary-key, unique and foreign-key failures to GORM sentinels. The caller owns and closes the underlying `*sql.DB`.

The separate `db/sqlite` migration helpers continue accepting a caller-owned connection without selecting a driver. See the [adapter documentation](db/sqlite/gorm/README.md) for usage and driver compatibility.

## MySQL with GORM

[`db/mysql`](db/mysql/README.md) opens a shared SQL/GORM pool with `New` or `NewWithConfig`, optional database creation, verified TLS and versioned SQL migrations. Application queries keep multi-statements disabled; migration methods use a separate pool. Existing caller-owned pools can use the standalone migration helpers.

## Development

Run `make dep`, `make build`, `make lint`, `make generate` and `make test` through the Docker development image. Tests use `-race`; `make test PKG=./db/mysql` runs the real MySQL suite alone. Test targets use the host Docker socket to start sibling testcontainers. Other targets do not receive the socket. `make test-coverage` runs the complete suite with a 10% minimum.

`make sec` scans source and Git history for secrets, dependency vulnerabilities and container configuration, and writes `sec.sarif`. CI uses the shared `code-workflow` to run the same Make commands and upload coverage and security reports. Tag jobs wait for a successful default-branch run of the same commit before publishing badges and a release. See [security checks](docs/security.md) for failure conditions and advisory notes.

Use `make pkg-add PKG=module@version`, `make pkg-add-tool PKG=module/cmd/tool@version`, `make pkg-update PKG=module`, `make pkg-upgrade`, or `make pkg-remove PKG=module` to change dependencies. These commands apply the release-age check, refresh checksums and vendor, and run security checks. `make help` lists the supported commands.
