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
