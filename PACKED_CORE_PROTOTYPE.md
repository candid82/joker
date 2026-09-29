# Packed core prototype

This branch adds a `packed_core` build tag. With the tag, core namespaces are emitted as packed top-level bytecode plus compact Var/linter summaries instead of as a statically initialized Go object graph. `joker.core` is loaded at startup and the other core namespaces remain lazy.

Build with:

```bash
go generate ./core
(cd core && gofmt -w a_*.go)
go build -tags packed_core -o joker-packed .
```

## Benchmark

Measured on an Apple M3 Pro (`darwin/arm64`) with Go 1.26.0. The baseline is commit `a9f634a0` on `vm`. Process timings are medians from interleaved runs with output redirected to `/dev/null`.

| Workload | Runs | Static core | Packed core | Change |
|---|---:|---:|---:|---:|
| `--version` | 80 | 10.25 ms | 14.65 ms | +43% |
| `-e nil` | 80 | 9.98 ms | 14.39 ms | +44% |
| require and invoke `joker.set` | 50 | 10.13 ms | 16.28 ms | +61% |
| lint empty CLJ file | 40 | 22.49 ms | 26.84 ms | +19% |
| lint `tests/linter/types-3/input.clj` | 20 | 23.58 ms | 28.02 ms | +19% |

A clean-cache `go build -a` was run twice in each order:

| Metric | Static core | Packed core | Change |
|---|---:|---:|---:|
| Build wall time | 6.80 s | 5.23 s | -23% |
| Build user CPU | 33.77 s | 30.76 s | -9% |
| Executable size | 27,640,210 B | 22,875,938 B | -17% |
| Median maximum RSS, `--version` (5 runs) | 32.9 MB | 37.9 MB | +15% |

The ten namespace images contain 2,420,923 bytes of bytecode and 152,937 bytes of summaries. Their quoted generated Go file is 9,002,702 bytes because arbitrary packed bytes still require substantial escaping.

## Verification

- `go test ./...`
- `go test -tags packed_core ./core/...`
- `go vet -tags packed_core ./...`
- `./all-tests.sh` using the packed-core executable

All passed.

## Conclusion

Packed core improves build time and executable size, but costs about 4.4 ms on every invocation, more when a lazy namespace is loaded, and increases peak startup RSS. The current static-core/packed-linter split remains preferable for a startup-sensitive CLI unless the size and build-time reductions are worth that runtime cost.
