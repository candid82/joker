# Joker VM implementation status

This is the living status and handoff document for the bytecode VM on the `vm` branch (code through `06dfe6e2`). For Day 11-specific measurements and optimization history, see [perf.md](perf.md); that file is not the VM feature tracker.

## How execution works today

- The VM is enabled by default for evaluation; `./joker --no-vm <file>` selects the AST evaluator for comparison. Parsing, macro expansion, linting, and formatting are not replaced by the VM.
- For eligible top-level expressions, `ProcessReader` compiles and runs bytecode. Otherwise it evaluates the AST and calls `CompileAST` to precompile eligible nested functions. Other reading/loading paths can still evaluate top-level AST expressions while compiling functions inside them (e.g. `load-string`).
- Eligible generated `joker.core` functions are compiled after startup, unless VM execution is disabled or linter mode is active. Currently **352 generated non-macro core functions** compile. Core macros (**52** at last inventory) remain parse-time macros rather than runtime bytecode functions.
- Compiled functions can call native procedures and AST-backed functions; unsupported functions keep their AST implementations. `IsVMCompatible`/`IsVMCompatibleFn` in `core/compile.go` are the admission gates, not a guarantee that every function is compiled: a compiler error also leaves it on AST. The `--vm-fallbacks` flag counts *VM calls into AST functions*, not execution starting in AST or calls into native procedures.

## Implemented and tested

- Bytecode execution for literals, local/global bindings and Vars, control flow, calls, closures, multi-arity and variadic functions, named self references, `loop`/function-level `recur`, throws, catches, and `try/finally`. Finally runs after normal completion, a caught or uncaught Joker error, and an error in a catch; nested finally blocks and cross-function throws are covered by parity tests. The parser still rejects `recur` across `try`.
- Literal vectors, maps, and sets use AST-compatible construction order and duplicate detection. Quoted list/set constants with safely serializable contents and direct `Var`/type literals are eligible. Captured values in initialized generated core closures can be embedded as constants; unresolved bindings still fall back.
- Compiled bytecode records source positions, including immutable native-call-site descriptors. Positions and call sites survive the internal packed-bytecode format. Error *wording* and stacktrace formatting may differ from AST; meaningful, correct source locations are required. The HTTP SSE forked test exercises an asynchronous error location.
- `core/vm_test.go` covers VM/AST parity for maps/sets, constants, packing, closures, `case`, Var identity, finally, source positions and calls. `tests/eval/vm_basic.joke` is also run in both modes. The current build passes `./run.sh --build-only`, `go test ./...`, `go vet ./...`, the eval suite with both VM and AST drivers, and the linter, formatter and flag suites.

## WIP / known boundaries

- **This is a hybrid evaluator**, not a VM-only interpreter. Standalone `CatchExpr` and unexpanded `MacroCallExpr` are not admitted by `IsVMCompatible`; macros normally expand during parsing and catches compile as parts of `TryExpr`. Other unlisted expression types and literals stay on AST. The list/set constant gate intentionally excludes metadata and nested objects whose identity, type or metadata the packer's print/read round trip cannot preserve (for example a list containing a Var). Quoted vector/map *objects* are not the same as vector/map *expressions* and are not generally admitted.
- Upvalues are currently copied into closed values when `OP_CLOSURE` runs. `Compiler.endScope` still has a TODO for captured-local cleanup. Ordinary closure tests pass, but mutations of captured slots, forward references and `letfn`-style cases merit targeted AST/VM tests before treating closure lifecycle as complete.
- VM exception dispatch handles Joker `Error` values. AST `try` also runs `finally` for non-`Error` Go panics; VM currently re-panics those without dispatching finally. Decide whether such host panics are within the VM's supported semantics, and test the decision.
- The internal packed-bytecode layout changed to carry source positions. Regenerate embedded data with `./run.sh --build-only` when changing bytecode/packing; there is no explicit packed-format version or compatibility check for old blobs. Legacy `OP_MAP`/`OP_SET` opcode handlers remain alongside the newer incremental literal opcodes.
- VM call frames and the value stack have fixed limits (`vmFramesMax`, `vmStackMax`). A pooled VM is returned to the pool only after successful `VMExecute`; panicking calls allocate a replacement on the next call. Benchmark or harden this only with exception-path parity tests.

## Next steps

1. Add focused parity tests for the closure/upvalue and non-`Error` exception boundaries above; fix any semantic discrepancy before widening compilation.
2. Inventory fallback sites on *other* workloads: Day 11 alone is not a coverage test. Add admissible AST expression/literal types only after runtime **and packed-bytecode** round trips preserve value, identity where observable, metadata and relevant source locations.
3. Version/validate the internal packed format if blobs must survive across builds. Keep linter, formatter, macro expansion and startup modes under regression tests.
4. Only after correctness checks, address overhead such as native call-site bookkeeping, exception-path pooling and sequence/call allocations. Record workload-specific numbers in [perf.md](perf.md), not here.

## Reproducing checks

```sh
./run.sh --build-only
go test ./...
go vet ./...
./eval-tests.sh
./joker --no-vm tests/run-eval-tests.joke
./linter-tests.sh && ./formatter-tests.sh && ./flag-tests.sh
./joker --vm-fallbacks path/to/program.joke  # exact total, sampled sites (1/256), report on stderr
```

A zero fallback count means no measured **VM-to-AST function calls** in that run; it does not mean all executed code ran in the VM. The eval harness's `--no-vm` driver still launches *forked* test subprocesses without that flag; run individual forked inputs with `./joker --no-vm` when needed. Always verify results under `--no-vm` when changing compilation eligibility.
