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
Each independent `Execution` owns one VM for its active lifetime; `vmPool`
caches VMs between independent roots. A synchronous callback from a native
call re-enters its paused caller's VM rather than taking another VM from the
pool. Independent `go` bodies and HTTP callbacks start new executions. Core
`apply` and `eval` also have optional execution-aware native entries.

Generated Go data contains compiled functions and compact summaries used for
linter inference; parsed forms are removed before the object graph is emitted.
Functions created by bytecode are compiled closures. A dynamically created
function that still holds a parsed source expression compiles and caches its
prototype on first invocation; compilation errors are surfaced rather than
falling back to AST execution.

The build-time code generator also executes bytecode when bootstrapping generated
object graphs. Parsing and build-time linter analysis use ASTs, but generated
core namespaces retain only bytecode and compact analysis results. There is no
AST runtime evaluator or fallback.

## Semantics and representation

- Lexical references resolve by parser `Binding` identity, not spelling.
- All arities of a closure share one capture layout.
- Ordinary lexical captures copy values. Closures made before `recur` retain their
  previous iteration's bindings. `letfn` allocates shared heap cells before any
  initializer runs, supporting forward references, shadowing, and escaped mutual
  recursion. No capture points into a VM's stack.
- Metadata is evaluated before its expression and applied to the result.
- Calls capture/check the callable before evaluating arguments. Mutable Vars are
  looked up at runtime. Guarded integer calls verify the captured implementation
  and its helper's current binding; `with-redefs` and Var rebinding work normally.
- Vectors, maps, and sets preserve representation, construction order, and duplicate
  detection. Runtime literal pools can hold arbitrary Joker
  objects, including objects passed through `eval`, without serialization.
- `try`/`catch`/`finally` handles normal results, Joker exceptions, catch failures,
  nested finally blocks, and native Go panics. Host panics execute finally but do
  not match Joker catch clauses. `recur` across `try` remains a parser error.
- Value, call-frame, handler, and pending-finally storage grow dynamically.
  Numeric instruction operands are checked unsigned 32-bit values, including
  local slots, captures, call counts, collection sizes, and jump distances.
- Normal VM returns no longer use panic/recover. Pooled VMs are cleaned up on both
  successful and exceptional exits; discarded values and frames are released.

## Guarded integer execution

`core/int_compile.go` joins machine-integer initializers and every `recur` backedge
by `Binding` identity, iterating to a fixed point. Lexical `let` facts propagate;
nested loop/function bodies have their own backedges. These are conditional facts
for an optimized path, not unchecked linter inference or promises about Vars.

The first specialization covers unary `inc` and binary `=`. Both instructions
check concrete operand types and recognize an exact compiled leaf wrapper. Its
private helper must still be the original pure Go procedure, with no
execution-aware override. Wrapper shape is cached from immutable bytecode; helper
bindings are checked after argument evaluation on every call. A failed guard uses
the ordinary captured-callee call without replaying effects. Non-integer values
from a rebound implementation can flow through the same loop and trigger further
generic calls; no separate loop clone or execution-context fallback is needed.

Fresh integer results use a private object-stack marker and parallel pointer-free
`[]int` payload storage (one additional machine word per allocated stack slot).
Local copies, temporaries, `recur`, and compiled returns preserve raw payloads.
Constants and preexisting objects retain their source info and original spelling.
Closures snapshot boxed values; native/unknown calls, collections, pending finally
results, public `Peek`/`Pop`, and root returns materialize ordinary `Int` objects.
Core native arguments are materialized before borrowing the slice. Both stack
arrays grow/reset together, including across callback reentry and GIL suspension.
This is not a replacement of the whole Object stack or a general numeric JIT.

## Diagnostics and native boundaries

Bytecode stores source positions and immutable call-site names. Runtime error
snapshots collect logical VM frames on demand, including native callback entries,
without per-call stacktrace allocation. Execution-context handles are invalidated
before a VM can be reused, so retained native contexts cannot access pooled storage.

The GIL serializes Joker work, but acquiring its mutex alone does not bind an
execution: another VM may have run while the current native call was doing I/O.
Native calls use `RT.Suspend()` before releasing the GIL and the token's
`Resume()` after reacquiring it, before invoking Joker or creating errors. This
restores the owning `RT.vm`, `RT.currentExpr`, and call stack, and rejects an
expired context. Independent Go events use `RT.LockIndependent()` to clear the
ambient state, then `CallIndependent()` for their Joker callback. These are
distinct boundaries; ordinary lazy-sequence realization can use the ambient
context of its *consumer* while holding the GIL, not its creator's VM.

`RT.vm` remains a GIL-protected ambient context, not the owner of a VM. The
HTTP SSE forked test checks that an asynchronous native error names the server
site rather than an unrelated concurrently running request. Synchronous caller
stacks, macro names, linter error locations, and packed source information also
have tests.

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
./linter-tests.sh
./formatter-tests.sh
./flag-tests.sh
go test ./core -run '^$' -fuzz FuzzVMPackedExpression -fuzztime=10s -parallel=2
```

Regression tests cover metadata and evaluation order, multi-arity captures,
recursive/shadowed bindings, callback reentry, Var rebinding, dynamic evaluation,
opaque constants, large operands/stacks/jumps, exceptions and host panics,
source traces, expired execution contexts, packed object sharing and metadata,
malformed/truncated packed data, and compilation of generated functions.

The bounded fuzzer compares direct and packed VM results. It generates only safe,
finite programs, not arbitrary source with access to filesystem/network procedures.

Future optimization should preserve the strict execution boundary and semantic
tests; extending arithmetic specialization must retain the implementation/type
guards, argument evaluation order, and boxing at escape boundaries.
Startup compilation, source-position storage, closure allocation, and native
callback costs remain useful profiling targets.
