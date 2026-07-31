# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v0.2.0 — 2026-07-31

New `scope` package, which absorbs the `slogging` context logger.

- **Added `scope`.** Carries attributes on a `context.Context` and stamps them
  onto the logger it hands back, so every line logged under that context is
  tagged without passing the values around. Two tiers, split by whether an
  attribute describes the process or the work:
  - `SetGlobal` / `RemoveGlobal` / `GetGlobal` — process-wide (build id, service
    name, region), never serialized.
  - `Set` / `Remove` / `Get` — per-context (request_id, user_id), and the only
    tier that crosses a process boundary.
  - `GetLogger` applies both, the context tier winning a key collision.
  - `ToJSON` / `FromJSON` move the context tier across a hop — a Temporal
    header, a queue message, an HTTP header, a subprocess env. A process fact
    put in that tier would overwrite the receiving service's own value and its
    logs would then name the wrong deploy, which is why the tiers are separate.
  - `Attr` constrains values to primitives that render sanely as both a slog
    attribute and JSON, so a struct fails to compile rather than producing
    something unreadable at runtime.
- Attributes are applied when a logger is requested, not when they are set.
  That is what makes `Remove` possible at all — `slog` offers no way to
  un-apply an attribute already built into a logger — and it means a logger
  fetched before a `Set` or `Remove` keeps what it was built with. Call
  `GetLogger` where you log rather than holding the result.
- **Breaking.** The `slogging` root package is removed; its `GetCtxWithLogger`
  and `GetLogger` now live in `scope`. Replace
  `slogging.GetLogger(ctx)` with `scope.GetLogger(ctx)` and
  `slogging.GetCtxWithLogger(ctx, l)` with `scope.GetCtxWithLogger(ctx, l)`.
  Two packages each exporting a `GetLogger`, only one of which carried the
  scope attributes, meant reaching for the wrong one compiled and ran and
  showed up only as a missing `request_id`. `slogging/loki` is unaffected and
  keeps its import path.
- `scope/.example` is a runnable walkthrough of both tiers and a process hop.

## v0.1.4 — 2026-07-27

- Added a GitHub Actions CI status badge to the README.

## v0.1.3 — 2026-07-27

Update golang.org/x/text and other dependencies to patched versions (resolves govulncheck findings).

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
