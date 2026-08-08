# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v0.3.5 — 2026-08-08

Finishes the SQL redaction v0.3.3 started. Two Postgres literal forms were still
reaching the log stream verbatim.

### Fixed

- **Escape strings (`E'...'`) leaked their tail.** Postgres escape strings use
  backslash escapes, so `\'` does not end the literal — but the scanner treated
  it as if it did and resumed emitting mid-secret. `SELECT E'secret\'tail'`
  logged `tail'`.
- **Dollar-quoted strings (`$tag$...$tag$`) leaked entirely.** They contain no
  quotes at all, so the scanner never entered redaction and copied the body
  straight through. `SELECT $payload$secret-value$payload$` logged
  `secret-value` verbatim — the exact failure v0.3.3 existed to close, still
  open for anything using Postgres' own quoting.

  Dollar-quote tags are validated rather than assumed: only a well-formed tag
  opens a literal, so a bare `$` in arithmetic is not mistaken for one.

### Changed

- Redaction and truncation are now a single pass with a byte budget, rather than
  normalising the whole statement and truncating afterwards. A 10 MB query used
  to allocate 10 MB to produce a 2 KB preview.
- `Trace` checks whether the record would be emitted before doing any of this
  work, so a disabled level costs nothing.

The trace field shape is unchanged (`sql_preview` / `sql_bytes` /
`sql_truncated`), and no exported signature moved.

### Docs

- The `scope/.example` header pointed at `github.com/psyb0t/slog-configurator`,
  which was renamed to `github.com/psyb0t/slogging` — the example itself stays
  dependency-free and leans on `slog.Default()`.

## v0.3.4 — 2026-08-08

Repository infrastructure only. No library code changed.

- Added the imported-by badge: a count of the public packages importing this
  module, linking to `importers.md` on the `badges` branch — the importing
  repositories, grouped, package counts descending, and flagged when the owner
  differs from this repo's.
- It measures **blast radius, not adoption**, and this repo is the sharpest case
  for that distinction: zero stars, and half the other modules here depend on it.
  Nobody stars a bag of shared building blocks; they just import it. The count is
  what tells you how much breaks when an exported name moves.
- **It will read `unknown` until pkg.go.dev crawls this module.** That is
  deliberate: "nothing imports this" and "I could not tell" are different facts,
  and rendering the second as a confident `0` would be worse than saying nothing.
- Refreshed weekly rather than daily, because pkg.go.dev's crawl lags
  publication by days and each run drags the full test suite along (the badges
  job needs the coverage artifact). The whole pipeline runs rather than a
  badges-only job: the badge publisher republishes only what a run produced, so
  a badge-only refresh would delete the coverage, version and license badges.
- The cron slot is derived from a hash of the repository name rather than
  chosen — GitHub cron has no randomness, and its scheduler sheds queued runs
  hardest at the round times a human would pick.

## v0.3.3 — 2026-08-07

The gorm logger stops putting raw SQL — values and all — into your logs.

### Fixed

- **`db.GormSlogLogger` no longer logs query literals.** Every traced statement
  went out as a `sql` field containing the query verbatim, so anything inlined
  into it — an email address in a `WHERE`, a token in an `INSERT`, a whole
  encrypted blob — was copied into the log stream and then into whatever
  ships it onward. Statements are now normalised before logging: single-quoted
  strings and numeric literals are replaced with `?`, and runs of whitespace
  collapse to one space.

- **A single statement can no longer blow up a log line.** The normalised query
  is truncated at 2048 bytes, backing off to a valid UTF-8 boundary so a
  multi-byte character is never cut in half.

  The trace fields change shape as a result: `sql` is replaced by
  `sql_preview` (the redacted, truncated statement), `sql_bytes` (the original
  length, so you can still see that a query was enormous) and `sql_truncated`.
  **Anything querying on the `sql` field needs repointing at `sql_preview`.**

### Added

- **`errors.ErrUnavailable`** — for a capability the process knows about but
  that was never wired: an optional dependency left unconfigured, a feature
  switched off, a subsystem that failed to start. Distinct from `ErrNotFound`
  on purpose: an empty lookup tells the caller to fix the query, an unavailable
  capability tells them to fix the configuration.

- `.gitignore` covers agent scratch directories, local env files, coverage and
  profile artifacts, and editor state.

## v0.3.2 — 2026-08-01

Infrastructure only. No Go code changed and the exported API is untouched —
every commit in this release is under `.github/workflows/`.

- The pipeline was split: building and publishing stay in `pipeline.yml`, and
  everything that leaves the host now lives beside it in
  `mirror-and-archive.yml`.
- The repository is mirrored to Codeberg as well as GitLab.
- It is archived to the Wayback Machine, Software Heritage, and archive.org.
- Issues opened on either mirror are copied back to GitHub every six hours, and
  the GitHub copy is closed when the original closes.
- Pull requests are switched off on both mirrors. They are force-pushed from
  GitHub, so anything merged on a mirror would be destroyed by the next sync.
  Issues and forking stay enabled.

## v0.3.1 — 2026-07-31

No change to the exported API — `scope`'s surface is byte-for-byte what `v0.3.0`
shipped.

- **A context now carries attributes and never a logger.** `v0.3.0` removed the
  exported `GetCtxWithLogger` but kept the mechanism as an unexported helper for
  the tests; that helper and the context key behind it are gone too, so
  `GetLogger` builds from `slog.Default()` unconditionally. Keeping a private
  path that only tests exercised meant the package under test behaved slightly
  differently from the package callers get — where output goes is slog's
  business, configured once at startup.
- The tests capture output by swapping `slog.Default()` for the duration of one
  test and restoring it after, instead of pinning a logger onto a context. That
  is process-wide state, so the tests that do it no longer run with `t.Parallel`
  — two of them swapping at once would each read the other's output. The suite
  is green under `-race`.

## v0.3.0 — 2026-07-31

- **Breaking.** `scope.GetCtxWithLogger` is removed. It pinned the logger that
  scoped loggers are built from, and outside this package's own tests nothing
  called it: `GetLogger` builds from `slog.Default()`, which is where slog
  configuration and any installed handler already live, so a context needs no
  preparation before `Set` or `GetLogger` work on it. The capability survives as
  an unexported helper the tests use to capture output per test without
  `slog.SetDefault` racing across `t.Parallel`.

  **This supersedes the migration note in v0.2.0 below**, which pointed
  `slogging.GetCtxWithLogger` at `scope.GetCtxWithLogger`. If you need to pin a
  logger to a single context rather than the process, configure the handler on
  `slog.Default()` at startup instead.

  The exported surface is now `Set` / `Remove` / `Get`, `SetGlobal` /
  `RemoveGlobal` / `GetGlobal`, `GetLogger`, `ToJSON` / `FromJSON` and `Attr`.

## v0.2.1 — 2026-07-31

- Upgraded `google.golang.org/grpc` from v1.79.3 to v1.82.1, resolving
  [GO-2026-6061](https://pkg.go.dev/vuln/GO-2026-6061). The vulnerable code was
  reachable from `temporal/client.go`, `temporal/workflow.go` and
  `utils/cryptoutil/certs.go`, so `govulncheck` failed the build rather than
  merely reporting it. Pulls `go.opentelemetry.io/otel` to v1.43.0 and the
  `genproto` API/RPC modules forward as transitive requirements; `vendor/` is
  regenerated to match.

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
