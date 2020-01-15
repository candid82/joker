<img src="https://user-images.githubusercontent.com/882970/48048842-a0224080-e151-11e8-8855-642cf5ef3fdd.png" width="117px"/>

# Developer Notes

These notes are intended for developers working on the internals of Joker itself. They are not comprehensive.

## Library Code (Namespaces)

As with Clojure, Joker supports "libraries" of code organized into _namespaces_. It offers a number of namespaces that are built-in to the Joker executable itself, as well as the ability to dynamically (at run time and on-demand) extend these namespaces via external Joker source files typically organized into directory trees and deployed alongside the Joker executable. (Currently, Joker does not support dynamic extension via non-Joker code, such as Go plugins.)

Whether built-in (as described below) or separately deployed via source files written in Joker (as described in [Organizing Libraries (Namespaces)](https://github.com/candid82/joker/LIBRARIES.md), developers should be aware of the progression of any given namespace.

## Namespace States

The states through which a given namespaces transitions are:

1. Available
2. Mapped
3. Loaded

### Available Namespaces

A namespace is _available_ if its source code is either:

* compiled (in some form) directly into the Joker executable
* deployed as Joker code such that a running Joker executable can locate and load it

The _compiled_ namespaces are also the _built-in_ namespaces, and are described below. These are _not_ necessarily "mapped"; the _core_ namespaces are mapped on-demand when first referenced. (`joker.core` is referenced immediately upon startup of the Joker executable; `joker.repl` is as well, when running Joker as a REPL.)

A namespace that is available, but not mapped, is not found via e.g. `(the-ns 'joker.hiccup)`. The set of _core_ namespaces is hardcoded (in any given Joker executable) in `joker.core/*core-namespaces*`.

Other built-in namespaces, the so-called _std_ namespaces, start out as both available _and_ mapped.

### Mapped Namespaces

A namespace is _mapped_ if it is present in `(all-ns)`, which enumerates all the namespaces mapped into the current (global) environment.

In this state, the namespace is "registered" (to coin a synonym) with the canonical Clojure namespace mechanism as implemented by Joker.

But the namespace itself hasn't yet necessarily been initialized. Only when that happens (potentially "lazily") is the namespace said to be _loaded_.

### Loaded Namespaces

When actually needed, via a `:require` clause in an `(ns ...)` specification, due to `(require ...)`, or (for an already-mapped namespace) directly as a symbol qualifier via e.g. `joker.some.namespace/somevar`, a namespace is _loaded_, meaning its internal code and data structures are fully initialized.

For example, running Joker with the `--verbose` option to observe some of the pertinent transitions:

```
$ ./joker --verbose
FindNameSpace: Lazily initialized joker.string
Welcome to joker v0.14.1. Use EOF (Ctrl-D) or SIGINT (Ctrl-C) to exit.
user=> (all-ns)
(joker.core joker.math joker.http joker.io joker.url user joker.crypto joker.filepath joker.os joker.string joker.yaml joker.repl joker.base64 joker.csv joker.hex joker.html joker.json joker.strconv joker.time)
user=> joker.core/*core-namespaces*
#{joker.tools.cli joker.test user joker.template joker.core joker.walk joker.set joker.repl joker.hiccup joker.better-cond}
user=> (the-ns 'joker.hiccup)
<repl>:2:10: Exception: No namespace: joker.hiccup found
Stacktrace:
  global <repl>:2:1
  core/the-ns <joker.core>:2316:18
user=> (use 'joker.hiccup)
FindNameSpace: Lazily initialized joker.html
nil
user=> (the-ns 'joker.hiccup)
#object[Namespace "joker.hiccup"]
user=> (all-ns)
(joker.core joker.math joker.http joker.io joker.url joker.hiccup joker.yaml joker.repl user joker.crypto joker.filepath joker.os joker.string joker.strconv joker.time joker.base64 joker.csv joker.hex joker.html joker.json)
user=>
```

First, note that `joker.string` is lazily initialized. This is due to running Joker as a REPL, because that automatically loads `joker.repl`, which in turn requires `joker.string`.

Continuing to the interactive portion of the session, note that `joker.hiccup` is not present in the output of the first `(all-ns)` invocation. That's because it is only _available_ (as confirmed by its presence, as a _core_ namespace, in `joker.core/*core-namespace*`), but it is not yet _mapped_ (as confirmed by its absence when trying `(the-ns 'joker.hiccup)`).

Then, `(use 'joker.hiccup)` explicitly loads that namespace, meaning that it is _located_ (in this case, being a _core_ namespace, it is located "internally"), _mapped_, and _loaded_ (initialized).

Further, because `joker.hiccup` requires `joker.html` (an already-mapped namespace), the latter is _loaded_ (lazily initialized).

`(all-ns)` then confirms that `joker.hiccup` has become _mapped_.

Note that, at present, there are no explicit tests for whether a namespace is _available_ (in the general sense, beyond what `joker.core/*core-namespaces*` shows regarding available _core_ namespaces), nor for whether one is _loaded_ (initialized).

These distinctions should be of little, if any, important to developers of Joker code, since these transitions are (largely) managed automatically on behalf of canonical Joker code. But such distinctions are potentially of interest to developers working on Joker internals, including _core_ or _std_ namespaces.

## Built-in Namespaces

As explained in [the `README.md file`](https://github.com/candid82/joker/README.md), Joker provides several built-in namespaces, such as `joker.core`, `joker.math`, `joker.string`, and so on.

All necessary support for these namespaces is included in the Joker executable, so their source code needn't be distributed nor deployed along with Joker. This allows Joker to be deployed as a stand-alone executable.

The built-in namespaces are organized into two sets:

* Core namespaces, which provide functions, macros, and so on necessary for rudimentary functioning of Joker

* Standard-library-wrapping ("_std_") namespaces, which provide Clojure-like interfaces to various Go standard libraries' public APIs

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

(Note that "when needed", as used above, is immediately upon startup for `joker.core`; it also applies to `joker.repl` when the REPL is to be immediately entered; otherwise, it applies when the namespace is referenced such as via a `require` or `use` invocation.)

As this approach does *not* involve the normal Read phase at Joker startup time, the overhead involved in parsing Clojure forms is avoided, in lieu of using (what one assumes would be) faster code paths that convert binary blobs directly to AST forms. Even _mapping_ of a core namespace does not occur until it is needed.

A disadvantage of this approach is that it requires changes to `core/pack.go` when changes are made to the AST.

#### Adding a Core Namespace

Assuming one has determined it appropriate to add a new core namespace to the Joker executable (versus deploying it as a separate `*.joke` file), one must code it up (presumably as Joker code, though some Go code can be added to support it as well).

Then, besides putting that source code in `core/data/*.joke`, modify these to pick it up during the build:

* **core/gen\_data/gen\_data.go** `files` array (after any core namespaces upon which it depends)
* **core/procs.go** `InitInternalLibs()` array

Further, if the new namespace depends on any standard-library-wrapping namespaces:

* Edit the **core/gen\_data/gen\_data.go** `import` statement to include each such library's Go code
* Ensure that code has already been generated (that library's `std/*/a_*.go` file has already been created)

(Do not add the namespace to `*loaded-libs*`; that's for only libraries that have already been loaded. It will be automatically added to `*core-namespaces*` as an "available" library; and, upon being loaded, it will be added to `*loaded-libs*`.)

Create suitable tests, e.g. in `tests/eval/`.

Finally, it's time to build as usual (e.g. via `./run.sh`), then run `./eval-tests.sh` or even `./all-tests.sh`.

Note that core libraries (other than `joker.core` and, when running the Repl, `joker.repl`) do not show up in `(all-ns)` until after they've been loaded via `:require` or similar.

### Standard-library-wrapping (std) Namespaces

These namespaces are also defined by Joker code, which resides in `std/*.joke` files.

(Note that, unlike with core namespaces, multi-level namespaces here would have pathnames reflecting multiple levels. E.g. a `joker.a.b` namespace would be defined by `std/a/b.joke`. However, such namespaces do not exist in Joker as of `v0.14.0`.)

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

The `a_*.go` files generated for _std_ namespaces cause the namespaces to be _mapped_ by the time the Joker executable has finished starting up. That's why they appear in `(all-ns)`, even when they haven't actually been loaded (lazily initialized).

#### Advantages and Disadvantages vis-a-vis Core Namespaces

As standard-library-wrapping namespaces are lazily loaded (i.e. on-demand), and needn't build up the ASTs that the core namespaces build up, they can be expected to offer lower overhead at startup and/or first-use time. That is, only namespace generation, interning of symbols, and metadata is built up; other logic is "baked in" via compilation of the Go code accompanying these namespaces.

However, any logic (such as conditionals, loops, and so on) to be performed by them must be expressed in Go, rather than Joker, code; this mechanism is designed for easier creation of "thin" wrappers between Joker and Go code, not as a general mechanism for embedding Joker code in the Joker executable.

Another advantage (besides performance) of this approach is that the resulting code that builds up the target namespace has no dependencies on any other Joker namespaces -- not even on `joker.core`.

That means a *core* namespace may actually depend on one of these (standard-library-wrapping) namespaces, as long as `std/generate-std.joke` has been run and the resulting `std/*/a_*.go` file has been made available in the working directory (e.g. by being added to the Git repository).

#### Optimizing Build Time

The `run.sh` script includes an optimization that avoids building Joker a second (final) time after it runs `std/generate-std.joke` to generate the `std/*/a_*.joke` files.

That optimization starts by computing a hash of the contents of the `std/` directory *before* running the script, and another one *afterwards*.

If the hashes are identical, `run.sh` assumes nothing has changed in the `std/*.joke` files with respect to the `std/*/a_*.joke` files present prior to running the script, and thus there's no need to rebuild the Joker executable.

(Of course, even if a `std/*.joke` file hasn't changed, any changes to `std/generate-std.joke` or any of the `std/*/*.go` files, handwritten or autogenerated, will result in a different hash being computed and thus a rebuild.)

#### Adding a New Standard-library-wrapping Namespace

Besides creating `std/foo.joke` with appropriate metadata (such as `:go`) for each public member (in `joker.foo`), one must:

* Add `'foo` to the definition of `namespaces` in **std/generate-std.joke**
* Add the namespace to `*loaded-libs*` by editing its `defonce` definition in `core/data/core.joke`
* `mkdir -p std/foo`
* `(cd std; ../joker generate-std.joke)` to create `std/foo/a_foo.go`
* If necessary, write supporting Go code, which goes in `std/foo/foo_native.go` and other Go files in `std/foo/*.go`
* Add the resulting set of Go files, as well as `std/foo.joke`, to the repository
* Add tests to `tests/eval/`
* Rebuild the Joker executable (via `run.sh` or equivalent)
* Run the tests (via `./all-tests.sh` or just `./eval-tests.sh`)

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

## Debugging Tools

### go-spew

When built via e.g. `go builds -tags go_spew`, the private `joker.core/go-spew` function is enabled. (Otherwise it does nothing and returns `false`.)

This function dumps, to `stderr`, the internal structure of the argument passed to it (i.e. a Joker object), and returns `true`.

Optionally, a second argument may be specified that is a map with configuration options as described in [the `go-spew` documentation](https://github.com/jcburley/go-spew), though not all such operations are yet supported by Joker's `go-spew` function.

For example, the internals of the keyword `:hey` can be output in this fashion:

```
user=> (joker.core/go-spew :hey {:MaxDepth 5 :Indent "    " :UseOrdinals true})
(core.Keyword) {
    InfoHolder: (core.InfoHolder) {
        info: (*core.ObjectInfo)(#1)({
            Position: (core.Position) {
                endLine: (int) 1,
                endColumn: (int) 24,
                startLine: (int) 1,
                startColumn: (int) 21,
                filename: (*string)(#2)((len=6) "<repl>")
            }
        })
    },
    ns: (*string)(<nil>),
    name: (*string)(#3)((len=3) "hey"),
    hash: (uint32) 819820356
}
true
user=>
```

*Note:* The `SpewState` configuration option is not currently supported; each distinct call to `go-spew` thus starts with a "fresh" state.
