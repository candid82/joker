# VM optimization experiments: AoC 2016 Day 11

The puzzle program is held unchanged. Candidates are ordered by estimated impact
from the initial CPU/allocation profile; estimates include indirect GC costs.

1. Small collection allocation: allocate persistent array-map/vector updates at
   their final size, avoiding clone-then-grow and excess retained capacity.
2. VM dispatch: continuous instruction loop; isolate native-call cleanup and
   reduce per-instruction overhead.
3. Closure captures: flatten per-capture heap objects into contiguous storage.
4. Sequence realization: eliminate intermediate sequence-node allocation while
   preserving lazy evaluation, memoization, and repeated traversal.
5. Collection hashing: traverse small collections without temporary sequences
   or map-entry vectors, preserving hash compatibility.
6. Native-to-VM callbacks: reduce entry/context allocation without weakening
   context expiry, exception handling, or callback traces.
7. Primitive boxing: avoid repeated Boolean boxing and same-type integer
   equality boxing; consider cached source-info-free small primitive values.

Each candidate is measured against the preceding retained version using the
original Day 11 input and all 30 cases in `benchmarks/run-benchmarks.joke`.
Measurements run sequentially with default GC settings. Report median Day 11
elapsed time and the geometric mean of per-case benchmark median ratios.
Negative percentages mean less time. Timing improvements must justify added
complexity; correctness checks are required before retaining a change.

Raw measurements, versioned binaries, and the measurement harness are stored in
`/tmp/joker-optimizations.a8qORf`.

| Optimization | Day 11 change | Benchmark geomean change | Decision |
| --- | ---: | ---: | --- |
| Final-size small collection allocation | +1.0% | -0.2% | Rejected: no meaningful improvement |
| Continuous VM dispatch loop | -7.5% | -10.4% | Kept: repeatable improvement across workloads |
| Flat closure captures | -1.8% | -1.7% | Kept: consistent small gain and simpler representation |
| Store realized sequence first/rest directly | -5.5% | -3.4% | Kept: removes intermediate nodes and preserves memoization |
| Direct small-collection hashing | -3.3% | +0.7% | Kept: clear Day 11 gain; benchmark change within observed noise |
| Allocate fresh callback contexts in batches | +0.05% | +0.2% | Rejected: no measurable benefit; avoid extra lifetime machinery |
| Primitive result boxing and integer equality | -4.9% | -5.0% | Kept: repeatable gains on both workloads |

Baseline: three Day 11 runs (21.19, 21.83, 21.63 seconds), median 21.63 s.
The first benchmark run had an outlier; a third baseline repeat makes the
per-case median robust to that sample. Subsequent candidates start with two
runs of each workload, with extra confirmation where needed.

## Combined result

The final retained version takes 16.99 and 17.18 seconds for Day 11 (median
17.09 s), compared with the 21.63 s baseline: **21.0% less elapsed time**.
Every run returns `31`. The geometric mean of all 30 benchmark time ratios is
**18.6% lower** than baseline. Individual experiment percentages above are
incremental and should not be added together.

The final allocation profile estimates **10.34 GiB in 204.8 million objects**,
versus 12.95 GiB in 295.5 million objects in the initial profile: approximately
20.2% fewer allocated bytes and 30.7% fewer objects. These are sampled cumulative
allocations, not peak memory. The final profiled execution also returns `31`
and takes 17.26 seconds. Profiles are saved as `final-cpu.pprof` and
`final-heap.pprof` in the experiment directory.

Most compute/sequence benchmark cases improve 14–27% overall. The largest final
regression is `(vec list-1000)` at 2.8%; the other largely unchanged vector-copy
cases range from -0.9% to +1.1%. These are short local measurements, not confidence
intervals. Differences near 1% are treated as noise, and no benchmark source or
iteration count was changed.

The final-size allocation experiment did not earn its keep even though it
reduced obvious redundant allocation paths. The callback experiment allocated
contexts in batches of 32, never reused a published handle, and passed repeated
expiry checks; it still provided no measurable speed benefit. Both source
changes were reverted. A larger same-VM callback redesign was not necessary to
evaluate this bounded allocation experiment and was not attempted.

## Validation and reproduction

Environment: Go 1.26.0, darwin/arm64. Starting commit:
`a9252f58375368b308a94ed59ffe78217b19b208` on `vm`.

Input: `/Users/candid/personal/advent/advent2016/day11/main.joke`, unchanged
SHA-256 `9c4a887024ee1d44f6502bb3c69f356db81338782f3e31e8c27477e4c0f10eba`.

Validation passed:

- `go test ./core` for every experimental implementation.
- `./run.sh --build-only`, including generation and `go vet ./...`.
- `go test ./...`, including native library tests.
- `./all-tests.sh`: evaluation (193 tests, 1196 assertions), flags, formatter,
  and linter suites.
- New tests cover lazy realization/memoization and retry after failure, direct
  hash compatibility with sequence hashing and hash maps, primitive source-info
  isolation, and mixed-number equality.

The raw JSON files in the experiment directory contain every per-case timing.
Each named binary corresponds to its candidate: `baseline`, `collections`,
`dispatch`, `captures`, `sequences`, `hashing`, `contexts`, and `boxing`.
`collections.patch` and `contexts.patch` preserve the rejected experiments
(the latter is cumulative relative to the starting source).

```sh
go build -o /tmp/joker-candidate .
/usr/bin/time -p /tmp/joker-candidate /Users/candid/personal/advent/advent2016/day11/main.joke
/tmp/joker-candidate benchmarks/run-benchmarks.joke --json
```

Repeat both workloads sequentially for each saved binary. Compare the median
Day 11 time and geometric mean of per-case median ratios; preserve the original
input, benchmark iteration counts, and default GC settings.
