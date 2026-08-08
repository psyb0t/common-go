# common-go

Shared Go building blocks for psyb0t services: error sentinels, HTTP helpers,
cache, DB, queues, rate limiting, structured logging, Temporal, and LLM
utilities.

Log scope propagation used to live here as `common-go/scope`. As of v0.4.0 it
has its own module — [`ctxscope`](https://github.com/psyb0t/ctxscope) — so it
can ship without waiting on gorm, echo, NATS or the Temporal SDK. Same API, new
package name.

[![CI](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml)
[![coverage](https://raw.githubusercontent.com/psyb0t/common-go/badges/coverage.svg)](https://github.com/psyb0t/common-go/actions/workflows/pipeline.yml)
[![version](https://raw.githubusercontent.com/psyb0t/common-go/badges/version.svg)](https://github.com/psyb0t/common-go/releases)
[![license](https://raw.githubusercontent.com/psyb0t/common-go/badges/license.svg)](LICENSE)
[![imported by](https://raw.githubusercontent.com/psyb0t/common-go/badges/importers.svg)](https://github.com/psyb0t/common-go/blob/badges/importers.md)
