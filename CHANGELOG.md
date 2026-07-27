# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v0.1.2 — 2026-07-27

Modernize to Go 1.26 + house lint/logging/error conventions.

- Go 1.26.
- `make lint` now runs `go fix -diff` first and fails if it finds unapplied fixes; `make lint-fix` applies `go fix` before `golangci-lint --fix`.
- Added a coverage badge (first badge in the README) wired through a new `coverage-percent.txt` value file produced by `make test-coverage` and consumed by the `badges` CI job.
- Replaced `logrus` with structured `log/slog` calls in `app-runner`.
- Extracted magic numbers in `temporal/wait.go` into named defaults.

## v0.1.1 — 2026-07-27

Add README status badges.

- Added self-hosted version and license badges (rendered as SVGs on the `badges` branch by the `create-badges` CI job, no third-party render service). Wired a badges job into pipeline.yml.
