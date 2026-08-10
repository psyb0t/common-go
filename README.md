# common-go

Shared Go building blocks for psyb0t services: error sentinels, cache, DB,
queues, rate limiting, Temporal, and LLM utilities.

Three packages here still ship but are superseded for new code — `errors` by
[`ctxerrors/commerr`](https://github.com/psyb0t/ctxerrors) (it now re-exports
those sentinels from there, aliasing the same values so `errors.Is` still
holds), `http` by
[`aichteeteapee`](https://github.com/psyb0t/aichteeteapee), and the Loki slog
sink `slogging/loki` by
[`slogging/handlers/loki`](https://github.com/psyb0t/slogging).

[![CI](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml)
[![coverage](https://raw.githubusercontent.com/psyb0t/common-go/badges/coverage.svg)](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml)
[![version](https://raw.githubusercontent.com/psyb0t/common-go/badges/version.svg)](https://github.com/psyb0t/common-go/releases)
[![license](https://raw.githubusercontent.com/psyb0t/common-go/badges/license.svg)](LICENSE)
[![imported by](https://raw.githubusercontent.com/psyb0t/common-go/badges/importers.svg)](https://github.com/psyb0t/common-go/blob/badges/importers.md)

> **Moved out:** log scope propagation used to live here as `common-go/scope`.
> Since v0.4.0 it is its own module — [`ctxscope`](https://github.com/psyb0t/ctxscope) —
> so it ships without waiting on gorm, echo, NATS or the Temporal SDK. Same core
> API, new package name (`scope.Set` → `ctxscope.Set`), plus a new `NewHandler`
> for automatic context-scope propagation. Pin `v0.3.x` to keep the old path.
