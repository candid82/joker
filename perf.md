# Joker performance findings: Advent of Code 2016, day 11

Workload: `/Users/candid/personal/advent/advent2016/day11/main.joke`

The program is intentionally an allocation-heavy breadth-first search. These findings focus on optimizing Joker rather than changing the program's algorithm or state representation.

## Profile summary

The current `master` build produced the correct result, `31`.

| Measurement | Result |
|---|---:|
| CPU-profiled wall time | 46.1 s |
| User CPU time | 137.8 s |
| Peak RSS | 773 MB |
| Estimated allocated space | 37.3 GB |
| Estimated allocated objects | 1.24 billion |

The CPU total exceeds wall time because Go's garbage collector runs concurrently on multiple cores. GC accounted for 56.5% cumulative CPU. This workload is primarily allocation/GC-bound, not hash-map-operation-bound.

Profiles from the investigation were written to:

- `/tmp/day11-full.cpu`
- `/tmp/day11-full.mem`

## Top three optimization opportunities

### 1. Eliminate per-call argument and `LocalEnv` allocations

The largest direct allocation sources were:

| Source | Allocated space |
|---|---:|
| `evalSeq` argument slices | 8.43 GB |
| `LocalEnv.addFrame` | 5.97 GB |
| `LocalEnv.addEmptyFrame` | 4.09 GB |
| `LocalEnv.replaceFrame` | 698 MB |

Relevant locations are `core/eval.go:283` and `core/parse.go:313-345`.

Every function call creates an argument slice and environment node. `let`, `loop`, and `recur` create further environment storage. Potential improvements include:

- specialized zero-, one-, two-, and three-argument call paths;
- evaluating arguments directly into preallocated activation slots;
- an evaluator-owned activation stack rather than heap-allocated linked environments;
- updating loop bindings in place for `recur`, while copying frames that escape into closures;
- longer term, compiling parsed expressions to a register- or stack-based VM.

Reducing these allocations should also substantially reduce GC work.

### 2. Remove `defer`-based call-stack bookkeeping from the hot path

`Fn.Call` currently pushes a Joker call frame and uses `defer RT.popFrame()` for every Joker function call. The defer at `core/object.go:746` allocated approximately 1.91 GB.

The hottest flat CPU symbol was:

```text
runtime.tryDeferToSpanScan  15.33 CPU seconds flat, 25.03% cumulative
```

`Eval` also installs a defer for every expression in order to restore `RT.currentExpr`.

Potential improvements include:

- explicit push/evaluate/pop on successful calls;
- restoring a saved call-stack depth at exception recovery boundaries;
- storing source positions in evaluator or VM frames so `RT.currentExpr` does not need to be changed and restored around every AST node.

Exception and `try`/`catch` semantics require care. Error creation already snapshots the runtime stack before unwinding, which should make a less defer-heavy design possible.

### 3. Make sequence pipelines allocation-efficient

`LazySeq.realize` accounted for about 22% cumulative CPU. Significant direct allocation sources included:

| Source | Allocated space |
|---|---:|
| `ArraySeq.Rest` cursor objects | 1.13 GB |
| `ArrayMap.Keys` | 1.13 GB |
| lazy-sequence creation | 764 MB |
| `FnExpr.Eval` closure creation | 991 MB |
| `ToSlice` | 256 MB flat / 3.07 GB cumulative |

The workload heavily exercises `for`, `mapcat`, `remove`, `filter`, `some`, and `every?`. Their recursive Joker implementations construct closures, lazy sequences, environments, argument slices, and sequence cursors.

Potential improvements include:

- Go-backed mapping, filtering, and concatenating sequence types;
- iterator-based reductions that do not allocate a new `Rest()` cursor per element;
- making `ArrayMap.Keys` and `Vals` return views/cursors rather than copied slices;
- changing `into` to use transients for editable targets.

## Lower-priority observation

Caching collection hashes initially appeared promising because states are keys in `frontier` and `visited`. However, collection hash/equality and persistent-map operations represented only roughly 3% of sampled CPU. Call frames, defer bookkeeping, and sequence allocation should be addressed first.

## VM branch investigation

The `vm` branch was merged with `master` and profiled with the same program. Its stack-based design directly addresses the first recommendation:

- `VM.stack` is a preallocated value stack;
- `VM.frames` is a preallocated call-frame stack;
- compiled locals occupy value-stack slots instead of linked `LocalEnv` objects;
- `OP_RECUR` copies new values into existing loop slots rather than allocating a replacement environment.

A fair run using the same VM-capable binary produced:

| Mode | Wall time | User CPU | Peak RSS |
|---|---:|---:|---:|
| `--no-vm` | 46.05 s | 138.82 s | 849 MB |
| VM | 42.19 s | 115.91 s | 846 MB |

This is an 8.4% wall-time improvement and a 16.5% reduction in user CPU. A separate CPU-profiled VM run took 43.06 seconds.

The allocation profile also improved, but it shows that only part of the workload currently reaches the VM:

| Measurement | AST on `master` | VM | Change |
|---|---:|---:|---:|
| Allocated space | 37.34 GB | 33.97 GB | -9.0% |
| Allocated objects | 1.239 billion | 1.058 billion | -14.6% |
| GC cumulative CPU | 56.5% | 53.2% | -3.3 percentage points |
| `evalSeq` allocations | 8.43 GB | 6.25 GB | -25.9% |
| `LocalEnv.addFrame` | 5.97 GB | 4.35 GB | -27.2% |
| `LocalEnv.addEmptyFrame` | 4.09 GB | 2.35 GB | -42.5% |

VM profiles were written to:

- `/tmp/day11-vm.cpu`
- `/tmp/day11-vm.mem`

### Why the VM does not improve this workload more

Much of `joker.core` is still represented by generated AST functions. Code generation deliberately makes `CompileAST` a no-op, so calls from compiled user code into functions such as the sequence helpers frequently fall back through `Fn.callAST`. The VM profile consequently still contains `evalSeq`, `LocalEnv`, `Eval`, and `LazySeq.realize` near the top.

The VM also introduces or retains several allocation sources:

- `VM.callValue` allocates an argument slice when calling an AST function or native `Callable` (1.26 GB);
- VM closure creation allocates a `Fn`, an upvalue slice, and separately boxed closed upvalues;
- `VM.run` uses `defer`/`recover`, and `Fn.callVM` retains call-stack `defer` bookkeeping;
- lazy-sequence implementations and their cursor allocations are unchanged.

The next VM work should therefore be:

1. Compile generated core functions to bytecode, or compile compatible core AST functions lazily and cache their prototypes.
2. Pass native call arguments as a view into the VM value stack, or add fixed-arity native call interfaces, instead of allocating `[]Object` in `VM.callValue`.
3. Keep captured upvalues open while closures share an active VM frame, and allocate closed storage only when a closure actually escapes.
4. Replace panic/defer-based normal VM returns and per-call stacktrace bookkeeping with explicit VM control flow and VM-native stacktrace frames.
5. Apply the separate sequence-pipeline improvements; the VM alone does not remove lazy-sequence and collection-cursor allocation.

Overall, the VM validates the first recommendation, but the current WIP implementation only partially covers the hot path. Extending VM coverage and removing its AST/native-call bridges should produce a much larger gain than the current 8%.

## Sequence optimization results

The sequence work was implemented on `master` in commit `2423406b` and then merged into `vm`. Timings below are consecutive unprofiled runs of the same workload; instruction counts are included because individual wall-time measurements have some scheduler noise.

| Step | Wall time | User CPU | Instructions | Incremental wall change |
|---|---:|---:|---:|---:|
| Baseline `master` | 46.12 s | 138.15 s | 1.120 T | — |
| Strided `ArrayMap.Keys`/`Vals` views | 46.25 s | 137.24 s | 1.110 T | +0.3% |
| Native sequence/set reduction | 40.81 s | 123.28 s | 0.988 T | -11.8% |
| Transient-backed `into` experiment | 41.48 s | 122.96 s | 0.991 T | +1.6% |
| Go-backed `map`/`filter`/`remove`/`mapcat` | 39.26 s | 94.51 s | 0.924 T | -5.4% |
| Native `some` and `every?` loops | 36.98 s | 89.25 s | 0.867 T | -5.8% |
| Revert transient-backed `into` | 36.71 s | 90.42 s | 0.859 T | -0.7% |
| Release realized transform sources | 32.78 s | 85.08 s | 0.754 T | -10.7% |
| Go-backed `concat` and final cleanup | 32.53–32.97 s | 85.29–85.92 s | 0.750 T | within ~1% CPU; wall noisy |

The transient `into` change was rejected because two runs showed no improvement (41.48 and 41.97 seconds) and slightly more instructions. All other steps were retained. Clearing a realized transforming sequence's references to its callable and input was especially important: it reduced peak RSS in the adjacent runs from 781 MB to 321 MB and greatly reduced GC scanning.

Compared with the original `master` baseline, the final AST implementation produced:

| Measurement | Before | After | Change |
|---|---:|---:|---:|
| Wall time | 46.12 s | 32.97 s | -28.5% |
| User CPU | 138.15 s | 85.92 s | -37.8% |
| Peak RSS | 728 MB | 327 MB | -55.1% |
| Retired instructions | 1.120 T | 0.750 T | -33.1% |
| `pprof` allocated space | 36.47 GB | 28.27 GB | -22.5% |
| Allocated objects | 1.239 billion | 0.890 billion | -28.1% |

Final AST profiles:

- `/tmp/day11-seq-final.cpu`
- `/tmp/day11-seq-final.mem`

### VM after sequence optimization

The sequence changes benefit both evaluators. With the optimized core merged into `vm`:

| Mode | Wall time | User CPU | Peak RSS | Retired instructions |
|---|---:|---:|---:|---:|
| `--no-vm` | 32.87 s | 86.02 s | 329 MB | 0.755 T |
| VM | 30.65 s | 74.21 s | 442 MB | 0.680 T |

Relative to the pre-optimization VM run, VM wall time fell from 42.19 to 30.65 seconds (-27.4%), user CPU fell by 36.0%, peak RSS fell by 47.7%, allocated space fell to 25,203 MB, and allocated objects fell from 1.058 billion to 0.708 billion (-33.1%). The VM's incremental advantage over optimized AST execution is now 6.8% wall time and 13.7% user CPU.

Final VM profiles:

- `/tmp/day11-vm-seq.cpu`
- `/tmp/day11-vm-seq.mem`

## VM generated-core and native-call follow-up

The VM branch now compiles the generated, closed `joker.core/into` function at startup (only with VM enabled) and calls built-in core `Proc` values with a borrowed VM-stack argument slice. AST functions and third-party procedures still receive owned argument slices because their arguments can escape.

Compiling **all** eligible generated core functions was not safe: 258 of 274 candidates compiled, but eval and linter tests failed (notably namespace loading). A tested allowlist of eight sequence functions passed the suites but increased Day 11's user CPU from about 72 to 85 seconds. Individually, compiling `group-by` was similarly slow. Compiling only `into` reliably reduced retired instructions by roughly 2.3% and peak RSS from about 442 to 342 MB; other individually tested helpers brought no clear benefit. Broad generated-core compilation needs semantic fixes and better VM-to-AST transitions before expanding the allowlist.

| VM configuration | Wall time | User CPU | Peak RSS | Instructions |
|---|---:|---:|---:|---:|
| Before both changes (fresh run) | 29.45 s | 72.60 s | 436 MB | — |
| Borrowed core-proc arguments, without generated-core compilation | 30.09 s | 72.68 s | 442 MB | 0.669 T |
| Plus compiled generated `into` (two runs) | 29.32–29.38 s | 72.47–72.59 s | 342 MB | 0.653 T |
| `--no-vm` after both changes | 33.04 s | 86.34 s | 327 MB | 0.755 T |

These runs are noisy: the combined wall-time difference from the initial run is within ~1%, though instruction count and RSS improve consistently. A memory profile after both changes reports 24,802 MB and 682.3 million allocated objects, versus 25,203 MB and 707.8 million before them. `VM.callValue` flat allocations fall from about 1,240 MB to 887 MB. The remaining allocations are largely AST-fallback calls, whose arguments cannot be borrowed from the reusable VM stack without copying escaping frames.

Follow-up profile: `/tmp/vm-final.mem`.

## Wider generated-core VM coverage

Bisection of the failing eval (`control.joke`) and linter (`unused-ns`) cases isolated `joker.core/with-bindings*`. The VM compiler had marked its `try/finally` as compatible, but `compileTry` skips `finally` on a normal exit and exception handling does not execute it on every exit path. That leaked temporary namespace bindings and caused subsequent parse errors. `IsVMCompatible` now rejects `try/finally`, retaining the AST path for such functions until the VM implements its complete semantics. Regression tests cover compatibility and restoration on normal and exceptional exits.

After that guard, compiling all eligible closed generated core functions (257 functions) passed eval and linter suites. This moves substantial portions of the sequence pipeline into the VM: `evalSeq` flat allocations fell from about 3,678 MB to 1,263 MB, and `LocalEnv.addFrame` from 3,376 MB to 1,166 MB. However, the new profile revealed an allocation regression: `OP_VECTOR` made a persistent vector with a 32-slot tail for every literal, while `VectorExpr.Eval` makes an `ArrayVector`. Matching AST literal construction removed approximately 7.9 GB of allocation in the all-core profile.

| VM batch | Wall time | User CPU | Peak RSS | Instructions | Allocated space | Objects |
|---|---:|---:|---:|---:|---:|---:|
| Prior VM (`into` only) | 28.54 s | 71.75 s | 340 MB | 0.653 T | 24,802 MB | 682 M |
| All compatible generated core, old `OP_VECTOR` | 26.44 s | 71.31 s | 333 MB | 0.621 T | 25,796 MB | 450 M |
| All compatible generated core, AST-equivalent `OP_VECTOR` | 24.30–24.67 s | 55.55–56.06 s | 322 MB | 0.533 T | 17,896 MB | 446 M |
| Same build, `--no-vm` | 32.66 s | 85.85 s | 325 MB | 0.755 T | — | — |

Relative to the fresh 29.45 s VM baseline before the generated-core and argument-slice work, the final VM is roughly 17% faster on Day 11. Against AST on the same build it is about 25% faster in wall time. No algorithm or state representation in the workload changed. The new Go tests, eval tests (190 tests/1175 assertions), linter, formatter and flag tests, and `go vet` pass.

Before/after profiles: `/tmp/vmcore-all.mem`, `/tmp/vmcore-final.mem`, `/tmp/vmcore-all.cpu`. Future work: implement complete VM `try/finally` handling and investigate remaining AST fallbacks.

## VM-to-AST fallback attribution and literal parity

`./joker --vm-fallbacks <file>` enables sampled attribution of VM `OP_CALL` invocations that enter an AST function. The report goes to stderr at exit, with the **exact total** call count and approximate per-site counts (one sample per 256 calls). Sites identify the calling compiled function (source line when available) and the callee function definition. The flag does not count calls originating entirely within AST code or native procedures. Normal execution does not collect samples. VM bytecode currently does not preserve source lines, so many caller sites appear as `<anonymous>` without a line number.

For Day 11, the report counted 20,234,917 VM-to-AST calls; its largest estimated sites were `core/assoc` (8.49 million), `core/conj` (7.31 million), `core/reduce` (3.01 million), and the workload's `near-floors` function (1.00 million). The complete report is in `/tmp/vm-attribution-final.err`. These are the next candidates for investigating *why* a function is not compiled; this flag itself does not change compilation.

Parity tests now cover empty maps/sets, map type selection at both sides of `HASHMAP_THRESHOLD`, key/value and set-element evaluation order, and duplicate detection including the timing of side effects. The VM previously built map/set literals only after evaluating all operands, reversed array-map insertion order, silently discarded duplicate entries, and used `Assoc` (which copied intermediate maps). New incremental literal opcodes mirror `MapExpr.Eval` and `SetExpr.Eval`; legacy `OP_MAP`/`OP_SET` remain for packed bytecode. Day 11 still returns `31`; an uninstrumented run after this change took 23.80 seconds (previous 24.30–24.67 seconds, within run-to-run variation).

## Compiling the hottest core fallbacks

Compiled named generated `conj` and `assoc` using their existing closed, self-only environments: the VM's slot 0 supplies the self binding. The compiler now treats function arities as `recur` targets, including variadic arities and nested loops. The generated `reduce` function requires a `Reduce` type literal, so that literal is permitted **only when checking `reduce` for core precompilation**; widening it globally changed error-position behavior in the HTTP SSE forked test. Unsupported constructs such as `try/finally` still fall back.

Day 11 (all runs returned `31`; times are single runs or small ranges, not statistically significant):

| Variant | VM→AST fallbacks | Wall time | User CPU | Peak RSS | Allocated (sampled) |
|---|---:|---:|---:|---:|---:|
| Before | 20.23M | 23.94s | 54.86s | 320MB | ~17.9GB (earlier profile) |
| `conj` compiled | 12.50M | 23.42s | 50.63s | 321MB | — |
| `assoc` also compiled | 3.97M | 21.97–22.46s | 46.22–46.40s | ~318MB | 14.02GB |
| `reduce` also compiled, **before** frame fix | 0.99M | 21.82–22.93s | 48.26–49.50s | ~317MB | 15.32GB |
| `reduce` compiled, **after** frame fix | 0.99M | 21.00–21.48s | 44.05–44.24s | ~317MB | 13.44GB |
| `--no-vm` | N/A | 31.72s | 85.13s | 327MB | — |

`reduce` alone initially increased allocations: callbacks from native reducers call compiled functions outside the running VM, and every `Fn.callVM` frame created a new empty `CallExpr` when no AST call site was available (around 1.96GB attributed to `Runtime.pushFrame`). Reusing one immutable empty call-site placeholder removed that allocation, and lowered both user CPU and wall time. Profiles: `/tmp/vm-assoc-only.mem`, `/tmp/vm-self-final.mem`, `/tmp/vm-reduce-shared.mem`. `recur` support also made 16 other previously excluded core functions eligible; compiled-core counts progressed from 257 before these changes to 274 with `conj`, 275 with `assoc`, and 276 with `reduce`. The remaining sampled fallback is predominantly the workload's `near-floors` at `/Users/candid/personal/advent/advent2016/day11/main.joke:27` (~0.99M calls), invoked from `next-states`.

## Broadening VM compilation safely

The `conj`/`assoc` allowlist and `reduce`-only type-literal gate are now removed. Named generated core functions compile when their bindings resolve; two pre-existing closed `comp` functions (`not-every?` and `not-any?`) now compile by capturing their initialized environment values in constants. `try/finally` executes on normal completion, matched catches, uncaught exceptions, exceptions thrown inside catches, and nested finally blocks; it preserves the prior result unless the finally itself throws. `recur` across `try` remains rejected by the parser as before. Incompatible or unresolved functions still fall back to AST. The core startup now precompiles **331** generated functions, versus 276 previously.

Bytecode carries source positions through compilation and packed-bytecode serialization (the internal packed format changed; regenerate embedded data with `./run.sh --build-only`). Native VM calls retain immutable call-site descriptors while running so error locations from asynchronous work remain meaningful. Stacktrace formatting need not match AST exactly; the HTTP SSE forked test now names actual source locations instead of `<file>:0:0`. Go VM-versus-AST parity tests cover normal/caught/uncaught/nested finally behavior, cross-function throws, captured environment values, packed positions, and error locations. Go tests/vet, eval suites in VM and AST modes, and linter/formatter/flag tests pass.

Day 11 still returns `31`: VM **22.06–22.55s**, ~**13.53GB** sampled allocations and **332–334MB** peak RSS, versus `--no-vm` **31.54–31.86s**, **326–327MB** peak RSS. Broader compilation and tracking source positions cost roughly 1–1.5 seconds of VM wall time on this workload compared with the narrow build, but remove policy-based restrictions independent of this particular benchmark. At this stage the remaining ~0.99M VM→AST calls targeted the workload's `near-floors` (line 27), called from `next-states` (line 42).

## Quoted collections and Var literals

`case` macro expansion puts quoted sets in `LiteralExpr`, and `reverse` uses an empty-list literal. `IsVMCompatible` previously rejected both even though `compileLiteral` emits `OP_CONST` and AST `LiteralExpr.Eval` returns the same constant on each call. It now admits lists and sets containing printable scalar elements or similarly safe nested lists/sets, while retaining AST fallback for collection metadata and elements whose types cannot safely round-trip through the packer's print/read format. Direct `*Var` literals are also accepted: the packer has a dedicated Var encoding that resolves to the same namespace binding, preserving identity. Quoted collections *containing* Vars remain excluded because printing a collection does not use that dedicated encoding.

Go parity tests check VM and AST values, types, repeated-call identity, packed-bytecode round trips, `case` dispatch, and exclusion of unsafe collection literals. All **352** generated non-macro core functions now compile (up from 331); 52 macros are intentionally left for parse-time expansion. Day 11 returns `31` in both modes: VM **21.39s**, **337MB** RSS; `--no-vm` **32.49s**, **326MB** RSS. Opt-in fallback attribution counted **zero** VM→AST calls on this run (previously ~0.99M). Other unsupported expression or literal types still retain AST fallback.

## Complete VM runtime (including macros)

The subsequent correctness/integration work removes runtime AST fallback entirely.
Generated functions compile on first invocation in every namespace, including
macros and native callbacks. Files, REPL, `eval`, and loading all use the VM.
The old eligibility gates, sampled-fallback flag, name-based arithmetic
substitution, fixed stack limits, and panic-based normal returns are no longer
part of runtime execution. See [VM.md](VM.md) for the implementation contract.

A fresh same-binary comparison, using `/usr/bin/time -l`, returned `31` in both modes:

| Mode | Wall time | User CPU | Peak RSS |
|---|---:|---:|---:|
| VM | 21.60 s | 44.02 s | 309 MiB |
| Reference AST (`--no-vm`) | 32.05 s | 85.58 s | 312 MiB |

These are single runs, not statistically significant comparisons with the earlier
21.39 s measurement. The VM is about 33% faster in wall time than the reference
on this run; completeness and semantic fixes did not sacrifice the previous
Day 11 advantage. No allocation profile was collected for this revision.

`./joker benchmarks/compare-vm.joke` also completed all 30 repository benchmarks.
Single-run VM wall times were 11–39% lower than the reference evaluator, including:

| Benchmark | AST | VM |
|---|---:|---:|
| `nested-let(5) x10000` | 1912 ms | 1294 ms |
| `fib(20)` | 1084 ms | 698 ms |
| `closure-test x10000` | 840 ms | 510 ms |
| `reduce(fn) x1000` | 157 ms | 117 ms |
| `arithmetic(42) x10000` | 1869 ms | 1365 ms |
| `sum-loop(10000)` | 5160 ms | 3435 ms |

The benchmark harness now executes in the selected evaluator as well; old comments
about deliberately preventing its VM compilation were removed. The reference
flag remains for development comparisons and generator bootstrap, not fallback.
