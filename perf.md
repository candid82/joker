# Joker performance work: Advent of Code 2016, days 11 and 12

Joker was profiled running:

- `/Users/candid/personal/advent/advent2016/day11/main.joke` (prints `31`).
- `/Users/candid/personal/advent/advent2016/day12/main.joke` (run from its directory to find `input.txt`; returns `{:ip 23, :regs {\a 318020, \b 196418, \c 0, \d 0}}`).

These are measurements of **Joker implementation overhead**, not recommendations to change either program. Profiles were collected with Joker's `--cpuprofile` and `--memprofile` options and inspected with `go tool pprof`. Timings below are unprofiled wall-clock runs on macOS/arm64 with Go 1.26.0; individual short runs are noisy, and cumulative CPU profile percentages include callees.

## Day 11

| Bottleneck | Attempt and outcome |
| --- | --- |
| Persistent collections and GC | Kept `ArrayMap.Assoc`'s single backing-array allocation and `ArrayMap.Without`'s exact-size copy in `core/array_map.go`. Sampled total allocations fell from **8.83 GB to ~8.20 GB** (about 7%). Three repeated unprofiled runs averaged **12.93s before vs 12.95s after**: no demonstrated wall-time improvement. |
| Lazy-sequence callback dispatch | Tried calling an already-compiled transform callback directly through the active VM instead of the generic callable dispatch. Day 11 timing stayed around **12.8s** in the initial comparison; reverted the change. |
| Closure allocations | Tried storing up to two captured values inline in `Fn` to avoid a separate upvalue slice. The larger `Fn` showed no speedup (the initial comparison was **12.83s before vs 12.91s after**); reverted the change. |

The initial profile showed roughly 8.5 GB of allocations, substantial GC CPU, and a hot `TransformSeq.realize`/callback path. These costs remain useful targets for future work, but the attempted callback and closure changes did not improve this workload.

## Day 12

| Bottleneck | Attempt and outcome |
| --- | --- |
| VM call dispatch | Kept a direct `OP_CALL` path for compiled functions of valid arity in `core/vm.go`; other calls retain the context-preserving path. Five isolated runs averaged **1.034s without this change vs 0.976s with it**. |
| Boxed integer arithmetic | Tried fast paths for `+`, `-`, `inc`, and `dec`. Returning Joker `Int` objects still required boxing, and neither allocation nor timing improved meaningfully; reverted the fast paths. |
| Type-test nil check | Kept a direct `Nil` type check instead of general `obj.Equals(NIL)` in `IsInstance` (`core/object.go`). It avoids numeric equality work on integers. Isolated allocation profiles measured **80.39 MB without vs 54.14 MB with** this change; five isolated runs averaged **0.998s without vs 0.976s with** it. |

Five unprofiled baseline runs averaged approximately **1.05s**, versus **0.99s** for the final version. The profiles of this short program also contained macOS `runtime.madvise` noise, which was not counted as a Joker optimization target.

## Validation

Both programs returned unchanged results. `core/array_map_optimization_test.go` checks persistent map independence and integer/nil type tests. `./all-tests.sh` (194 eval tests, 1,208 assertions, plus flag/formatter/linter suites), `go test ./...`, and `go vet ./...` passed.
