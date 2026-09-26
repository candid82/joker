# Joker bytecode evaluator

## Execution model

Bytecode is the default evaluator for **all runtime code, including macros**.
`Evaluate` / `TryEvaluate` in `core/execution.go` are the production entry points:

```
read → parse (execute macros in the VM) → compile → execute bytecode
```

Files, command-line expressions, the REPL/socket REPL, `eval`, `load-string`, and
library loading use this path. Top-level forms are still processed sequentially:
one form can establish macros or namespace bindings needed to parse the next.
Native procedures and callable collections retain their existing Go interfaces.
Callbacks from native procedures execute compiled functions on a pooled VM.

Generated Go data contains parsed functions and their initialized environments.
On first invocation, **every** such function compiles and caches its prototype,
regardless of namespace, whether it is a macro, or how it is reached (including
multimethods, lazy sequences, and native callbacks). A compiler error is surfaced;
there is no runtime AST fallback or per-namespace compilation allowlist.
Functions created by bytecode are already compiled closures.

`--no-vm` remains a development/test oracle, not a fallback mechanism. `Eval` and
`Fn.callAST` reject calls unless that mode was explicitly selected. The build-time
code generator also deliberately uses the reference evaluator to bootstrap the
generated object graphs. Parsing and linter inference still use ASTs; replacing
runtime evaluation does not require removing those representations. Packed linter
initialization remains bytecode even when the reference mode is selected.

The obsolete `--vm-fallbacks` flag and opportunistic `CompileAST` pass were removed.
Default-mode tests cannot silently fall back to AST evaluation.

## Semantics and representation

- Lexical references resolve by parser `Binding` identity, not spelling.
- All arities of a closure share one capture layout.
- Ordinary lexical captures copy values. Closures made before `recur` retain their
  previous iteration's bindings. `letfn` allocates shared heap cells before any
  initializer runs, supporting forward references, shadowing, and escaped mutual
  recursion. No capture points into a VM's stack.
- Metadata is evaluated before its expression and applied to the result.
- Calls check callability before evaluating arguments, as the reference evaluator
  does. Mutable Vars are looked up at runtime; name-based arithmetic substitution
  was removed, so `with-redefs` and Var rebinding work normally.
- Vectors, maps, and sets preserve reference-evaluator representation, construction
  order, and duplicate detection. Runtime literal pools can hold arbitrary Joker
  objects, including objects passed through `eval`, without serialization.
- `try`/`catch`/`finally` handles normal results, Joker exceptions, catch failures,
  nested finally blocks, and native Go panics. Host panics execute finally but do
  not match Joker catch clauses. `recur` across `try` remains a parser error.
- Value, call-frame, handler, and pending-finally storage grow dynamically.
  Numeric instruction operands are checked unsigned 32-bit values, including
  local slots, captures, call counts, collection sizes, and jump distances.
- Normal VM returns no longer use panic/recover. Pooled VMs are cleaned up on both
  successful and exceptional exits; discarded values and frames are released.

## Diagnostics and native boundaries

Bytecode stores source positions and immutable call-site names. Runtime error
snapshots collect logical VM frames on demand, including native callback entries,
without per-call stacktrace allocation. Execution-context handles are invalidated
before a VM can be reused, so retained native contexts cannot access pooled storage.

The runtime still uses Joker's existing GIL/global diagnostic context. Exact
caller stacks in asynchronous native errors can depend on scheduling. The HTTP
SSE forked test requires the exact error/source-location prefix rather than an
incidental stack from another suspended computation. Synchronous caller stacks,
macro names, linter error locations, and packed source information have tests.

Core native procedures may borrow argument slices for the duration of their call;
other native callables receive owned slices. A core procedure that retains its
argument slice must copy it. Escaped Joker closures never retain that slice.

## Internal packed format

Packed bytecode is internal, rebuildable build output, **not a stable public ABI or
an untrusted-code sandbox**. The header has a format marker/version; stale blobs
are rejected with a regeneration error.

Persistence is separate from compilation. `core/constant_pack.go` structurally
encodes supported portable constants, including collection metadata/source info,
shared object references, concrete collection types, and canonical Var, Type, and
Namespace references. Numeric encodings preserve numeric types and precision.
Opaque host/runtime objects and cyclic constants are explicitly rejected by the
packer, but remain valid in runtime constant pools.

The decoder checks counts and truncation and validates instruction boundaries,
constant/capture/subfunction operands, jump targets, stack states, exception
entries, and call-site tables before executing packed prototypes. Disassembly
uses the same instruction widths. Obsolete opcode handlers and duplicated legacy
single-arity serialization were removed.

After changing bytecode or packing, regenerate embedded data:

```sh
./run.sh --build-only
```

## Validation

```sh
./run.sh --build-only
go test -count=1 ./...
go vet ./...
./eval-tests.sh
./eval-tests.sh --no-vm
./linter-tests.sh
./formatter-tests.sh
./flag-tests.sh
go test ./core -run '^$' -fuzz FuzzVMExpressionParity -fuzztime=10s -parallel=2
```

The AST eval-test command uses `tests/joker-ast.sh` for forked tests and passes that
wrapper to tests which launch further interpreter subprocesses, so engine
selection propagates beyond the driver process.

Regression tests cover metadata and evaluation order, multi-arity captures,
recursive/shadowed bindings, callback reentry, Var rebinding, dynamic evaluation,
opaque constants, large operands/stacks/jumps, exceptions and host panics,
source traces, expired execution contexts, packed object sharing and metadata,
malformed/truncated packed data, and rejection of runtime AST entry.

The bounded differential fuzzer compares reference AST, direct VM, and packed VM
results. It generates only safe, finite programs, not arbitrary source with access
to filesystem/network procedures. A 10-second run completed about 89,000 cases
without a mismatch. Existing eval suites pass in both modes (193 tests, 1196
assertions, plus forked cases).

## Deliberately retained development infrastructure

The AST evaluator and `--no-vm` are retained for differential testing and generator
bootstrap. Removing them is a separate simplification, not necessary for complete
VM runtime coverage. Future optimization should preserve the strict execution
boundary and semantic tests; particularly, any arithmetic specialization must
respect mutable Vars. Startup compilation, source-position storage, closure
allocation, and native callback costs remain useful profiling targets.
