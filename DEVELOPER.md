<img src="https://user-images.githubusercontent.com/882970/48048842-a0224080-e151-11e8-8855-642cf5ef3fdd.png" width="117px"/>

# Developer Notes

These notes are intended for developers working on the internals of Joker itself. They are not comprehensive.

## Namespaces

As explained in [the `README.md file`](https://github.com/candid82/joker/README.md), Joker provides several built-in namespaces, such as `joker.core`, `joker.math`, `joker.string`, and so on.

All necessary support for these namespaces is included in the Joker executable, so their source code needn't be distributed nor deployed along with Joker. This allows Joker to be deployed as a stand-alone executable.

The built-in namespaces are organized into two sets:

* Core namespaces, which provide functions, macros, and so on necessary for rudimentary functioning of Joker

* Standard-library-wrapping namespaces, which provide Clojure-like interfaces to various Go standard libraries' public APIs

The mechanisms used to incorporate these namespaces into the Joker executable differ substantially, so it is important to understand them when considering adding (or changing) a namespace to the Joker executable.

### Core Namespaces

Core namespaces, starting with `joker.core`, define the features (mostly macros and functions) that are necessary for even rudimentary Joker scripts to run.

Their source code resides in the `core/data/` directory as `*.joke` files.

Not every such file corresponds to a single namespace; the `linter_*.joke` files modify the `joker.core` namespace, while the remaining files do correspond to namespaces, and are named by dropping the `joker.` prefix and changing all `.` characters to `_`. So, for example, the `joker.tools.cli` namespace is defined by `core/data/tools_cli.joke`.

When Joker is built (via the `run.sh` script), `go generate ./...` is first run. Among other things, this causes the following source line (a Go comment) in `core/object.go` to be executed:

```
//go:generate go run gen_data/gen_data.go
```

That builds and runs `core/gen_data/gen_data.go`, which defines, in the `files` array, a list of files (in `core/data/`) to be processed.

As explained in the block comment just above the `var files []...` definition, the files must be ordered so any given file depends solely on files (namespaces) defined above it (earlier in the array).

Processing a `.joke` file consists of reading the file via Joker's (Clojure-like) Reader, "packing" the results into a portable binary format, and encoding the resulting binary data as a Go source file named `core/a_*_data.go`, where `*` is the same as in `core/data/*.joke`.

As this all occurs before the `go build` step performed by `run.sh`, the result is that that step includes those `core/a_*_data.go` source files. The binary data contained therein is, when needed, unpacked and the results used to construct the data structures into which Joker (Clojure) expressions are converted when read (aka the Abstract Syntax Tree, or "AST").

(Note that "when needed", as used above is immediately upon startup for `joker.core`; also for `joker.repl` when the REPL is to be immediately entered; otherwise, when the namespace is referenced such as via a `require` or `use` invocation.)

As this approach does *not* involve the normal Read phase at Joker startup time, the overhead involved in parsing Clojure forms is avoided, in lieu of using (what one assumes would be) faster code paths that convert binary blobs directly to AST forms.

A disadvantage of this approach is that it requires changes to `core/pack.go` when changes are made to the AST.

#### Adding a Core Namespace

Assuming one has determined it appropriate to add a new core namespace to the Joker executable (versus deploying it as a separate `*.joke` file), one must code it up (presumably as Joker code, though some Go code can be added to support it as well).

Then, besides putting that source code in `core/data/*.joke`, modify these to pick it up during the build:

* **core/gen\_data/gen\_data.go** `files` array (after any core namespaces upon which it depends)
* **core/procs.go** `InitInternalLibs()` array

Further, if the new namespace depends on any standard-library-wrapping namespaces:

* Edit the **core/gen\_data/gen\_data.go** `import` statement to include each such library's Go code
* Ensure that code has already been generated (that library's `std/*/a_*.go` file has already been created)

At this point, building Joker would make the new namespace available at run time via e.g. `:require` in an `ns`, but not preloaded nor shown in `*loaded-libs*`.

(To add the namespace to `*loaded-libs*`, edit its `defonce` definition in `core/data/core.joke` and rebuild.)

Create suitable tests, e.g. in `tests/eval/`.

Finally, add the corresponding `(require 'joker.x)` line to `generate-docs.joke`, to get documentation generated automatically.

Now it's time to build as usual (e.g. via `./run.sh`), then run `./eval-tests.sh` or even `./all-tests.sh`.

### Standard-library-wrapping Namespaces

These namespaces are also defined by Joker code, which resides in `std/*.joke` files.

(Note that, unlike with core namespaces, multi-level namespaces here would have pathnames reflecting multiple levels. E.g. a `joker.a.b` namespace would be defined by `std/a/b.joke`. However, such namespaces do not exist in Joker as of `v0.13.0`.)

These `*.joke` files, however, have code of a particular form that is processed by the `std/generate-std.joke` script (after an initial version of Joker is built). They cannot, as explained below, define arbitrary macros and functions for use by normal Joker code.

#### The Joker Script That Writes Go Code

The `std/generate-std.joke` script, which is run after the Joker executable is first built (by `run.sh`), reads in the pertinent namespaces, currently defined via `(def namespaces ...)` at the top of the script.

(This should probably be changed to dynamically discover all the `*.joke` files in `std/`. See [the `gostd` fork's version](https://github.com/jcburley/joker/blob/gostd/std/generate-std.joke) for a sample implementation.)

`(apply require namespaces)` loads the target namespaces, then the script processes each namespace in `namespaces` by examining its public members and "compiling" them into Go code, which it stores in `std/*/a_*.go`, where `*` is the same name.

For example, `std/math.joke` is processed such that the resulting Go code is written to `std/math/a_math.go`.

*Note:* This processing does *not* handle arbitrary Joker code! In particular, "logic" (such as `(if ...)`) in function bodies is not recognized nor handled; it's actually discarded, in that it does not appear (in any form) in the final Joker executable. Similarly, no macros (public or otherwise) appear at all; so, as with logic in functions, they're useful only insofar as they might affect how other public members are defined during the running of `std/generate-std.joke`.

Instead, the processing consists primarily of examining the metadata for each (public) member and emitting Go code that, when built into (the soon-to-be-rebuilt) Joker executable, creates the namespace (`joker.math` in the above example), "interns" the public symbols, and includes (attached to those symbols) both suitable metadata and Go-code "stubs" that handle Joker code referencing a given symbol and the underlying Go implementation (typically a standard-library API, such as `math.sin` for `joker.math/sin`).

Those stubs handle arity, types, and results.

Whether they call Go code directly, or call support code written in Go (typically included in a file named `std/*/*_native.go`, e.g. `std/math/math_native.go`) -- and the specific Go-code invocation used -- is determined via the `:go` metadata and return-type tags for the public member, as defined in the original `std/*.joke` file.

#### Advantages and Disadvantages vis-a-vis Core Namespaces

As standard-library-wrapping namespaces are lazily loaded (i.e. on-demand), and needn't build up the ASTs that the core namespaces build up, they can be expected to offer lower overhead at startup and/or first-use time. That is, only namespace generation, interning of symbols, and metadata is built up; other logic is "baked in" via compilation of the Go code accompanying these namespaces.

However, any logic (such as conditionals, loops, and so on) to be performed by them must be expressed in Go, rather than Joker, code; this mechanism is designed for easier creation of "thin" wrappers between Joker and Go code, not as a general mechanism for embedding Joker code in the Joker executable.

Another advantage (besides performance) of this approach is that the resulting code that builds up the target namespace has no dependencies on any other Joker namespaces -- not even on `joker.core`.

That means a *core* namespace may actually depend on one of these (standard-library-wrapping) namespaces, as long as `std/generate-std.joke` has been run and the resulting `std/*/a_*.go` file has been made available in the working directory (e.g. by being added to the Git repository).

#### Optimizing Build Time

The `run.sh` script includes an optimization that avoids building Joker a second (final) time after it runs `std/generate-std.joke` to generate the `std/*/a_*.joke` files.

That optimization starts by computing a hash of the contents of the `std/` directory *before* running the script, and another one *afterwards*.

If the hashes are identical, `run.sh` assumes nothing has changed in the `std/*.joke` files with respect to the `std/*/a_*.joke` files present prior to running the script, and thus there's no need to rebuild the Joker executable.

#### Adding a New Standard-library-wrapping Namespace

Besides creating `std/foo.joke` with appropriate metadata (such as `:go`) for each public member (in `joker.foo`), one must:

* Add `'foo` to the definition of `namespaces` in **std/generate-std.joke**
* `mkdir -p std/foo`
* `(cd std; ../joker generate-std.joke)` to create `std/foo/a_foo.go`
* If necessary, write supporting Go code, which goes in `std/foo/foo_native.go` and other Go files in `std/foo/*.go`
* Add the resulting set of Go files, as well as `std/foo.joke`, to the repository
* Add tests to `tests/eval/`
* Rebuild the Joker executable (via `run.sh` or equivalent)
* Run the tests (via `./all-tests.sh` or just `eval-tests.sh`)

While some might object to the inclusion of generated files (`std/*/a_*.joke`) in the repository, Joker currently depends on their presence in order to build, due to circular dependencies (related to the bootstrapping of Joker) as described below.

### Beware Circular Dependencies

There's actually a circular dependency between the two sets of namespaces:

* `core/gen_data/gen_data.go` imports `std/string`
* `std/string/a_string.go` is generated by `std/generate-std.joke`
* `std/generate-std.joke` is run by the first Joker executable built by `run.sh`
* That Joker executable cannot be built until after `gen_data.go` has been run

This circular dependency is avoided, in practice, by ensuring that any `std/*/a_*.go` files are already generated and present before any new dependencies upon them are added to `gen_data.go`.

However, a `std/*.joke` file therefore cannot depend on any `core/data/*.joke`-defined namespace that, in turn, requires `gen_data.go` to import its `std/*/a_*.go` file.

So, while `joker.repl` and `joker.tools.cli` currently depend on `joker.string`, `std/string.joke` does not depend on them, and preexisted their being added to the core namespaces.
