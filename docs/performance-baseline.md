# Performance Baseline (2025-10-19)

Commands executed from repository root (`GOCACHE` pointed at `.gocache` to avoid
permission issues).

## Benchmarks

```bash
go test ./... -bench=. -run=^$ -benchmem
```

All benchmarkable packages completed (no benches in most packages). Individual
package benchmark suites finished within milliseconds; total wall time
≈25s including compilation.

### Executor 500-Step Benchmark

Synthetic benchmark added in `internal/infrastructure/engine/executor_bench_test.go`
to simulate a 500-step sequential pipeline:

```bash
go test ./internal/infrastructure/engine -run=^$ -bench BenchmarkExecutorPipeline500Steps -benchmem
```

Result:

```
BenchmarkExecutorPipeline500Steps-8    244    4.72ms/op    1.14MB/op    16002 allocs/op
```

This provides the baseline for future executor optimisations (T249/T250).

## Build Time

```bash
/usr/bin/time -f 'build time: %E' env GOCACHE=.gocache go build ./...
```

- Result: **1.52s** (build time: 0:01.52)
- Additional run with isolated `GOMODCACHE` failed due to sandboxed network
  access; the default module cache already contains dependencies so the primary
  timing stands.

## Domain Test Duration

```bash
go test ./internal/domain/... -count=1
```

- `internal/domain/pipeline`: 0.004s
- `internal/domain/plugin`: 0.002s
- Combined: **≈6ms** (<100ms requirement met).

## Full Suite Runtime

```bash
go test ./...
```

- Wall time: **≈3.6s**
- Serves as the new baseline for SC-008 (suite <20% slower than baseline); no
  previous baseline available, so future runs should stay within +20% of 3.6s.

## Notes

- Benchmarks currently limited; consider adding targeted benchmarks for new hot
  paths if needed.
- All commands succeeded under local caches; remove `.gocache`/`.gomodcache`
  directories after use if space becomes a concern.
