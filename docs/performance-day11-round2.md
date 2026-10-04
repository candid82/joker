# Day 11 optimization experiments: second round

This round starts with all retained changes from the first round. The puzzle
program and the 30 existing benchmarks remain unchanged.

Fresh profile: 17.11 seconds elapsed, 10.47 GiB allocated. GC marking accounts
for 34.3% of sampled CPU. Array-map cloning allocates 1.16 GiB; array-vector
cloning 0.66 GiB. VM context assignment on every opcode accounts for 0.42 sampled
CPU-seconds including its children; call-site map lookup accounts for about
0.30 seconds. `ArrayVector.Seq` and `String.Seq` allocate 0.40 and 0.17 GiB;
`ToSlice` allocates 0.30 GiB including traversal.

New candidates, in estimated impact order:

1. Transient map accumulation in `group-by`, avoiding repeated persistent map
   copies while keeping group values persistent. Also test a native bulk helper
   if eliminating copies alone is insufficient.
2. Transient accumulation in `into`, reducing repeated persistent updates in
   bulk collection construction, preserving target metadata and empty inputs.
3. Restore VM context at execution/call boundaries rather than every opcode.
4. Use indexed call-site descriptors instead of a Go map in the VM call path.
5. Read `first` directly from strings/vectors, avoiding temporary sequence nodes.
6. Materialize counted argument sequences directly and let known builtin
   callables borrow stack arguments, avoiding temporary slices and traversal.

Each candidate is tested and measured against the preceding retained version.
Use median Day 11 time and the geometric mean of per-case benchmark median
ratios. Run workloads sequentially with default GC settings; repeat uncertain
results. Negative percentages mean less time. Estimates are not confidence
intervals, and cumulative profile costs overlap.

Artifacts: `/tmp/joker-round2.SUs6Yl` (saved binaries, JSON measurements,
measurement harness, and CPU/allocation profiles).

| Optimization | Day 11 | Benchmark geomean | Decision |
| --- | ---: | ---: | --- |
| Transient `group-by` in Joker | +1.6% | +1.4% | Rejected: no gain |
| Native `group-by` builder | -19.9% | +0.5% | Kept: large Day 11 gain, benchmarks essentially unchanged |
| Transient `into` | -0.3% | -1.8% | Rejected: negligible Day 11 gain; metadata compatibility adds complexity |
| VM context restoration at call boundaries | -1.7% | -1.5% | Kept: removes per-opcode writes; tests cover native-call interleaving and panic cleanup |
| Indexed call-site descriptors | -2.7% | -0.3% | Kept: Day 11 gain survives paired repeat; packed format unchanged |
| Direct `first` for strings/vectors | -0.5% | +0.7% | Rejected: no meaningful gain in confirmation runs |
| Counted argument materialization | -0.4% | +0.01% | Rejected: no meaningful gain against contemporaneous control |
| Borrow stack arguments for builtin callables | +0.4% | +0.2% | Rejected: no gain; retain the existing ownership path |

## Timing controls

The saved indexed-call-site binary rose from a 13.14 s median to 13.76 s when
rerun later, with benchmark geomean rising 2.6%. This is environmental drift,
not a source change. The counted-argument experiment initially appeared 4.3%
slower than the early control, but differed by only -0.4% against the later
control; the latter comparison is reported. The direct-`first` decision is
likewise based on a confirmation run against that later control, not its
initial apparent regression. Small percentages remain uncertain.

## Implementation decisions

The retained `group-by` helper moves the bulk operation into Go, uses a private
transient map, and grows private small group vectors without persistent clones.
It preserves normal array-vector/tree-vector promotion, encounter order, empty
input behavior, and independently owned arguments for foreign key functions.
Merely replacing `assoc` with `assoc!` in the Joker implementation did not help;
the native helper additionally eliminates VM reducer dispatch and private
small-vector clones.

The transient `into` prototype saved little Day 11 time and exposed existing
metadata differences between collection types (including sets and map
promotion). Matching those differences would require more guards/fallbacks,
so the prototype was reverted instead of changing public behavior.

VM context restoration now happens at execution/call boundaries, including
panic cleanup, rather than every opcode. Indexed call descriptors trade a
larger static pointer table for removing a map lookup on every call. The
serialized representation is unchanged; absent descriptors, source names,
positions, and invalid descriptors are covered by tests.

All rejected runtime implementations are reverted. Their saved binaries and
measurements remain in the artifact directory. Compatibility tests for argument
ownership and sequence behavior are retained even where the fast path was not.

## Combined result

Three alternating confirmation pairs compare the saved round-two baseline
(all retained first-round changes) with the fully rebuilt final binary:

| Version | Day 11 runs (seconds) | Median |
| --- | --- | ---: |
| Round-two baseline | 18.00, 18.04, 18.13 | 18.04 s |
| Final retained version | 13.87, 14.11, 14.00 | 14.00 s |

The retained changes reduce Day 11 elapsed time by **22.4%** and the geometric
mean of all 30 benchmark time ratios by **2.2%**. Individual experiment changes
are incremental and must not be added together. The original fresh baseline
was 17.17 s; use the interleaved final comparison above for the cumulative claim,
since later runs of unchanged binaries also became slower.

The largest benchmark improvements are `sum-loop(10000)` (-5.7%),
`(seq (zipmap (range 8) (range 8)))` (-5.2%), `(vector)` (-4.7%), and
`closure-test` (-4.4%). The
four small regressions are vector/list copying cases, ranging from +0.5% to
+1.5%. These are local timing estimates, not statistical confidence intervals.

The final profile estimates **9.37 GiB allocated in 195.2 million objects**,
versus 10.47 GiB in 208.4 million objects in the fresh starting profile:
**10.5% fewer allocated bytes and 6.3% fewer objects**. These are sampled
cumulative allocations, not peak memory. Array-map cloning falls from 1184 MiB
to 513 MiB, and array-vector cloning from 678 MiB to 341 MiB. The final profiled
run also returns `31` and takes 14.01 s.

GC marking still accounts for 38.0% of sampled CPU, and total allocation remains
large. Removing VM work makes GC a larger fraction of the remaining profile;
the speedup should not be attributed solely to allocation reduction. Remaining
large allocation sites include VM closures/context entry, array-map removal,
and sequence traversal. Profiles are saved as `final-cpu.pprof` and
`final-heap.pprof` alongside their matching `final` binary.

## Validation and reproduction

Environment: Go 1.26.0, darwin/arm64, branch `vm`, starting commit
`a9252f58375368b308a94ed59ffe78217b19b208`, plus the retained first-round changes.

Input: `/Users/candid/personal/advent/advent2016/day11/main.joke`, unchanged
SHA-256 `9c4a887024ee1d44f6502bb3c69f356db81338782f3e31e8c27477e4c0f10eba`.
Benchmark source and iteration counts are unchanged. Every measured puzzle
execution is checked for the answer `31`.

Validation passed:

- `go test ./core` for each experimental implementation.
- `./run.sh --build-only`, including generation and `go vet ./...`.
- `go test ./...`, including native-library tests.
- `./all-tests.sh`: 194 evaluation tests, 1208 assertions, and all flag,
  formatter, and linter suites.
- `git diff --check` and formatting checks.

New tests cover grouping order, nil/false keys, string inputs, empty inputs,
large maps/groups and vector promotion, foreign callback argument ownership,
native-call context interleaving with normal return and panic, packed source
positions/names, invalid call descriptors, sequence copying/ownership, and
builtin/foreign callable argument semantics.

Most candidates have two runs of each workload; context restoration and
indexed call sites have three. The last three candidates use the refreshed
indexed-call-site control (`late-control`). The final combined comparison uses
three alternating baseline/final pairs, independent of the candidate timings.
All raw per-case measurements are in the corresponding JSON files beside each
saved binary in the artifact directory.

```sh
go build -o /tmp/joker-candidate .
/usr/bin/time -p /tmp/joker-candidate /Users/candid/personal/advent/advent2016/day11/main.joke
/tmp/joker-candidate benchmarks/run-benchmarks.joke --json
/tmp/joker-candidate --cpuprofile /tmp/day11-cpu.pprof --memprofile /tmp/day11-heap.pprof /Users/candid/personal/advent/advent2016/day11/main.joke
go tool pprof -top /tmp/joker-candidate /tmp/day11-cpu.pprof
go tool pprof -top -sample_index=alloc_space /tmp/joker-candidate /tmp/day11-heap.pprof
```
