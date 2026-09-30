# Joker performance work: Advent of Code 2016, days 11, 12, and 14

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

## Days 11 and 12 validation

Both programs returned unchanged results. `core/array_map_optimization_test.go` checks persistent map independence and integer/nil type tests. `./all-tests.sh` (194 eval tests, 1,208 assertions, plus flag/formatter/linter suites), `go test ./...`, and `go vet ./...` passed.

## Day 14, part one (native `take` pass)

Profiled `/Users/candid/personal/advent/advent2016/day14/main.joke` without changing the program. It prints 64 index/hash pairs, with final key index **25427**. Measurements used Go 1.26.0 on an Apple M3 Pro (macOS/arm64). The initial five unprofiled runs averaged **2.525s**.

### Top three bottlenecks and potential optimizations

These costs overlap; cumulative percentages must not be added together. The baseline CPU profile contained 2.51s of samples; allocation figures are sampled `alloc_space`, not retained heap size.

| Bottleneck | Baseline evidence | Potential optimization |
| --- | --- | --- |
| Interpreted lazy `take` and per-item objects | Closures/captures in `OP_CLOSURE`: **339 MiB**; `LazySeq`: **183 MiB**; cons nodes: **176 MiB**. The program repeatedly filters a 1,000-item lookahead, making this path much more important than hashing. | **Implemented:** a memoized native `take` traversal, replacing per-element closures, lazy wrappers, cons cells, and interpreted recursion with one sequence node. |
| VM instruction/callback dispatch | `VM.executeLoop`: **17.9% flat**, **71.7% cumulative**; `readOperand`: **5.6% cumulative**. `TransformSeq.realize` accounts for **53.4% cumulative**, including its callbacks and lazy source traversal. | Reduce interpreted instructions and callback entries via native sequence drivers (as done for `take`); consider operand decoding/instruction fusion separately. The direct callback bypass and larger inline-capture `Fn` attempted for day 11 did not demonstrate a speedup, so neither was reinstated. |
| Std native-call argument copying and boxing | The std-call branch of `VM.callValue`: **120 MiB**; `VM.PopN`: **88 MiB**; `string/includes?`'s Boolean result: **36 MiB**. This path is dominated by repeated substring predicates, not MD5. | Add an explicit opt-in borrowed-argument contract for non-retaining std procedures and use canonical boxed booleans. Do not borrow slices for arbitrary host callables: they may retain arguments. Not implemented in the part-one pass; part two below implements direct `Proc` dispatch without changing argument ownership. |

### Implementation

- `core/data/core.joke`: retain the outer `lazy-seq` and positive-bound check, delegating traversal to private `take-seq__`.
- `core/procs.go`, `core/procs_slow_init.go`: register the native helper. Realize its first node inside the outer lazy thunk so a source exception leaves that wrapper unrealized and retryable, just as before.
- `core/seq.go`: add a `transformTake` driver to the existing `TransformSeq`. Reuse its cached-first slot for the remaining numeric bound before realization, without enlarging map/filter/concat nodes. Check the bound before touching the source, use the existing numeric operations, and memoize first/rest only after successful realization.
- Regenerated ignored core build artifacts with `./build.sh`; no tracked std generated files changed.

### Measured impact

Seven **alternating-order, unprofiled before/after pairs**, using saved binaries built from the original and optimized sources, produced byte-identical stdout on every run:

| Metric | Before | After | Change |
| --- | ---: | ---: | ---: |
| Day 14 mean wall time | 2.543s | 1.509s | **40.7% less time; 1.685× speedup** |
| Day 14 median wall time | 2.540s | 1.503s | |
| Day 14 wall-time range | 2.529–2.571s | 1.491–1.551s | |
| Sampled total allocations | 1,103 MiB | 667 MiB | **39.5% less** |
| `BenchmarkTakeIndexed` mean (three runs, 1,000 items) | 485.5 µs/op | 67.8 µs/op | **86.0% less time** |
| Same benchmark bytes/op | 400,424 | 176,649 | **55.9% less** |
| Same benchmark allocations/op | 9,010 | 3,010 | **66.6% fewer** |

The optimized CPU profile reduced `executeLoop` flat samples from **450ms to 130ms** and cumulative samples from **1.80s to 0.80s**. These short profiles contain substantial macOS runtime/GC noise; the unprofiled paired timings are the speedup evidence. Remaining targets include native predicate dispatch/allocations and the native `take` nodes and boxed numeric decrements themselves.

Reproduction commands (save the pre-change executable separately for comparisons):

```sh
./build.sh
./joker --cpuprofile /tmp/day14.cpu --memprofile /tmp/day14.mem \
  /Users/candid/personal/advent/advent2016/day14/main.joke > /tmp/day14.out
go tool pprof -top ./joker /tmp/day14.cpu
go tool pprof -top -alloc_space ./joker /tmp/day14.mem
go test ./core -run '^TestTake' -count=1
go test ./core -run '^$' -bench '^BenchmarkTakeIndexed$' -benchmem -count=3
# Time without profiling; alternate saved before/after binaries, checking stdout.
/usr/bin/time -p ./joker /Users/candid/personal/advent/advent2016/day14/main.joke
```

Profiles, saved binaries, output snapshots, and paired timing samples from this session are under `/tmp/joker-day14-perf/` (temporary, not committed). The day 11/12 figures above remain historical: attempted additional cross-checks timed out, so no new comparisons are claimed for those workloads. The current day 12 script initializes `c` to 1, unlike the earlier workload whose result had `c` equal to 0.

### Day 14 validation

`core/take_seq_test.go` covers empty/short/infinite sources, zero/negative bounds without source realization, fractional/BigInt/BigFloat bounds, Unicode strings, nil/false elements, memoized tails, metadata/type/equality/hash, delayed argument errors, and source-exception retries at both the head and tail. Its public-`take` allocation-budget test failed before the change (**2,315 objects for 256 items**) and passes afterward (**778 objects**, budget 1,200).

Day 14 stdout is unchanged. `go test -count=1 ./...`, `go vet ./...`, and `./all-tests.sh` passed (195 eval tests, 1,219 assertions, plus flag/formatter/linter suites).

## Day 14, part two (updated workload)

The user updated the same external `main.joke` to perform **2,017 MD5/hex rounds per hash**. This materially changes the profile: part one's 40.7% timing improvement must not be applied to part two. The new output still contains 64 keys; the final index is **22045**.

### Top three bottlenecks and potential optimizations

The original saved Joker binary was profiled against the updated script: **22.74s profiled wall time**, **22.26s CPU samples**, and **12.10 GiB sampled allocations**.

| Bottleneck | Evidence | Optimization / next step |
| --- | --- | --- |
| Std procedure dispatch and allocation | Passing the value-typed `Proc` to `callOtherCallable(Callable, ...)` boxes a copy on the heap: **4.29 GiB**, **35.4% of allocations**. Owned argument slices add **1.49 GiB**. `callOtherCallable` accounts for **30.3% cumulative CPU**, including actual native work. | **Implemented:** handle std `Proc` directly in `VM.callValue`, eliminating the interface conversion and indirect generic dispatch. Continue allocating owned argument slices; no unsafe borrowing contract is introduced. A future opt-in borrowing API could target the remaining slice allocations. |
| MD5 and hex work / intermediate strings | `crypto/md5.block`: **9.8% flat CPU**; crypto wrapper **17.8% cumulative**; hex wrapper **6.0% cumulative**. MD5 wrapper allocates **1.72 GiB**, hex wrapper/encoding **2.43 GiB**. | Audit wrapper result boxing and hex output construction. A native digest-plus-hex or stretching API could remove intermediate strings/crossings, but would require changing the workload to use it; not done here. The MD5 compression work itself is inherent to part two. |
| VM loop and boxed numeric increments | `executeLoop`: **10.8% flat**, **70.8% cumulative CPU**, overlapping the native work above. `IntOps.Add`: **1.46 GiB**, mainly stretching-loop increments. | Reduce instruction/dispatch overhead or investigate canonical boxing for common integer results. Merely adding integer arithmetic fast paths did not help day 12; do not repeat that approach or bypass mutable Vars. Native `take` removes some interpreted work but now contributes only a small part of total runtime. |

### Implementation and measurements

`core/vm.go` now dispatches both core and std `Proc` values directly. Core procedures still borrow stack arguments; std procedures still receive the independently owned `PopN` slice and have their call operands popped before native entry. `InExecution` behavior, native-depth tracking, source-site restoration, and generic host-callable dispatch are unchanged. No std definitions or generated std files were modified.

Three **rotated-order, unprofiled rounds** compared all three saved binaries on the updated, unmodified script; every run produced byte-identical output:

| Version | Mean wall time | Individual runs |
| --- | ---: | --- |
| Original Joker | **22.729s** | 22.554, 22.900, 22.732s |
| Native `take` only | **21.870s** | 21.719, 21.801, 22.089s |
| Native `take` + direct `Proc` dispatch | **19.301s** | 19.294, 19.250, 19.358s |

- Combined improvement: **15.1% less wall time**, **1.178× speedup**.
- Dispatch change isolated against the `take`-only binary: **11.7% less wall time**.
- Native `take` isolated against the original binary: **3.8% less wall time** for part two.
- Sampled total allocations: **12.10 GiB → 7.35 GiB**, approximately **39.3% less**. The allocation profile confirms the `Proc`-boxing site is gone; owned argument slices remain at approximately 1.47 GiB.
- `BenchmarkVMStdProcDispatch` (three-run means): **38.3 → 22.3 ns/op**, **64 → 16 B/op**, **2 → 1 allocations/op**. The argument and callee are preboxed outside the timed loop to isolate dispatch overhead.

The profiling/reproduction commands in the part-one section now exercise part two because the external script changed. Current artifacts are `/tmp/joker-day14-perf/part2-{before,final}.{cpu,mem,out}`, `part2-times.json`, and the saved `joker-before`, `joker-after` (`take` only), and `joker-final` binaries.

### Part-two validation

`core/vm_proc_dispatch_test.go` adds a red-capable allocation test: before the dispatch change, std calls allocate two objects and fail its one-allocation budget; afterward they allocate only the owned argument slice. It also checks retained arguments survive VM stack reuse, and core calls retain their zero-allocation dispatch path. Existing host ownership, callback/reentry, native error/context, and arity tests pass.

The final build passes `go test -count=1 ./...`, `go vet ./...`, `git diff --check`, and `./all-tests.sh` (195 eval tests, 1,219 assertions, plus flag/formatter/linter suites). The updated day 14 output is byte-identical to the baseline.

## Day 14 follow-up: cached integer arithmetic results

Baseline is `ee8a4f71`, including native `take` and direct `Proc` dispatch. The intervening hex-wrapper experiment was reverted by the user because it did not demonstrate an end-to-end improvement. This pass leaves the external part-two program unchanged and does not alter bytecode, instruction dispatch, or std wrappers.

### Implementation

- `core/boxed.go`: expand the bounded integer cache from **0–255 to 0–2047**. Store its already-boxed values as `Number` interfaces and return `Number` from `boxInt`, allowing numeric operations to return a shared box directly. Existing `Object` callers still receive the same concrete `Int` values.
- `core/numbers.go`: `IntOps.Add` and `IntOps.Subtract` now return `boxInt(result)`. This covers integer `inc`, `dec`, `+`, and `-` without changing their dispatch. Unlike the previously rejected arithmetic fast paths, these results do not allocate a new box when cached.
- Negative and out-of-range results still allocate normal `Int` objects. Overflow, mixed numeric types, arbitrary-precision operations, equality/hash/identity, mutable Var rebinding, and reader/source information retain their prior behavior. Only fresh runtime results use the cache.
- The larger cache adds approximately **84 KiB of persistent storage on 64-bit systems** (interface entries plus integer boxes). The representation conversion makes an already-cached `count` microbenchmark approximately **0.8 ns slower**; this tradeoff is included in the full-program measurements below.
- Rebuilt/regenerated with `./run.sh --build-only`. No tracked std generated files changed.

### Allocation evidence and isolated arithmetic benchmarks

The new native-arithmetic allocation tests failed before the change: a result of 2016 allocated **one 32-byte object** for `inc`, `+`, `dec`, and `-`. They now allocate **zero**. Addition was implemented and measured before enabling subtraction.

Three-run means of `BenchmarkIntegerCacheArithmetic`, with operands preboxed and results escaped:

| Native operation | Before ns/op | After ns/op | Before → after B/op | Before → after allocs/op |
| --- | ---: | ---: | ---: | ---: |
| `inc`, result 2016 | 20.5 | 7.7 | 32 → 0 | 1 → 0 |
| `dec`, result 2016 | 20.5 | 7.7 | 32 → 0 | 1 → 0 |
| `+`, result 2016 | 22.4 | 9.7 | 32 → 0 | 1 → 0 |
| `-`, result 2016 | 21.0 | 8.0 | 32 → 0 | 1 → 0 |
| `inc`, result 4096 | 20.8 | 21.0 | 32 → 32 | 1 → 1 |
| `+`, result 4112 | 22.6 | 22.9 | 32 → 32 | 1 → 1 |
| `count`, result 1 | 2.39 | 3.19 | 0 → 0 | 0 → 0 |

Day-14 sampled allocation profiles confirm the intended mechanism:

| Profile | Total sampled allocations | Integer addition result allocations | Integer subtraction result allocations |
| --- | ---: | ---: | ---: |
| Baseline | 7,608 MiB (7.43 GiB) | 1,443 MiB | 78.5 MiB |
| Addition cache only | 6,193 MiB (6.05 GiB) | ~1 MiB | 74 MiB |
| Addition + subtraction cache | 6,118 MiB (5.97 GiB) | ~1 MiB | No samples attributed to `IntOps.Subtract` |

Combined total allocation reduction: approximately **19.6%**. Remaining uncached addition results include stream indices above the cache range. Totals are sampled and unrelated allocation sites vary between profiles; the per-operation zero-allocation tests provide the deterministic evidence.

### End-to-end impact

Five **reordered, unprofiled runs per version** compared saved baseline, addition-only, and combined binaries. Every completed run produced byte-identical stdout, with final key index **22045**. Collection resumed after a harness timeout; the incomplete run is not included.

| Version | Mean | Median | Individual runs |
| --- | ---: | ---: | --- |
| Baseline | **19.198s** | 19.257s | 18.944, 19.257, 19.299, 19.193, 19.297s |
| Addition cache only | **18.008s** | 18.189s | 18.189, 18.264, 17.300, 17.996, 18.293s |
| Addition + subtraction cache | **18.141s** | 18.149s | 18.155, 18.149, 18.271, 18.024, 18.107s |

The final combined version uses **5.5% less mean wall time**, a **1.058× speedup**. Its slowest run (18.271s) was faster than the fastest baseline run (18.944s). Unlike the reverted hex-wrapper attempt, this is a demonstrated full-workload improvement.

Addition caching provides the main benefit. The subtraction cache improves its isolated operations and eliminates another allocation source, but these full-workload measurements do **not** establish an additional wall-time improvement over addition alone; the addition-only mean also includes a faster 17.300s run.

### Validation and reproduction

`core/integer_cache_test.go` covers cache bounds and out-of-range allocation behavior, source-info/original-spelling isolation, signed integer overflow, mixed/precision-promoting numeric operations, public VM counter loops, equality/hash/identity, and Var rebinding. `core/boxed_test.go` extends primitive source-info isolation checks through the new cache range and its upper edge.

```sh
./run.sh --build-only
go test ./core -run '^Test(IntegerArithmeticCache|BoxedPrimitiveInfoIsolation)' -count=1
go test ./core -run '^$' -bench '^BenchmarkIntegerCacheArithmetic' -benchmem -count=3
./joker --cpuprofile /tmp/day14.cpu --memprofile /tmp/day14.mem \
  /Users/candid/personal/advent/advent2016/day14/main.joke > /tmp/day14.out
# Compare saved binaries without profiling; reorder runs and check stdout.
```

`go test -count=1 ./...`, `go vet ./...`, `go test -race ./core -count=1`, `git diff --check`, and `./all-tests.sh` passed (195 eval tests, 1,219 assertions, plus flag/formatter/linter suites). Artifacts are under `/tmp/joker-integer-cache-perf/`: saved `joker-{before,add,both}` binaries, workload snapshot, stdout snapshots, CPU/allocation profiles, three benchmark reports, `times.json`, and the suite log.
