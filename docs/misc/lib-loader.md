<img src="https://user-images.githubusercontent.com/882970/48048842-a0224080-e151-11e8-8855-642cf5ef3fdd.png" width="117px"/>

# Library Loader Behavior

Joker's library (namespace) loader (`joker.core/load`), used by `(ns ... :require ...)` and related macros and functions, normally assumes that namespaces' names correspond to their location in the filesystem. (See [Organizing Libraries (Namespaces)](../../LIBRARIES.md) for how to change this behavior.)

Specifically, the last part of the namespace (after the last dot) should match the file name (without the `.joke` extension), and the preceding parts correspond to the path to the file (with dots separating directories).

## Sample Namespace Layout

For example, say you have a directory named `ttt`  with the following structure:

```
├── core.joke
└── utils
    ├── a.joke
    └── b.joke
```

And here is the content of `*.joke` files:

```clojure
;; core.joke
(ns ttt.core
  (:require [ttt.utils.a :refer [a]]))

(a)

;; utils/a.joke
(ns ttt.utils.a
  (:require [ttt.utils.b :refer [b]]))

(defn a []
  (println "I am A")
  (b))

;; utils/b.joke
(ns ttt.utils.b)

(defn b []
  (println "I am B"))
```

Then you can run `core.joke` (regardless of the current directory or location of the Joker executable) and get the following (expected) output:

```
I am A
I am B
```

This mechanism is similar to how class loading works on JVM (and how namespace loading works in Clojure).

Please note that current working directory doesn't normally come into play here: Joker resolves libraries' paths relative to the file currently being executed. However, this behavior can be overridden via the `joker.core/*classpath*` and `joker.core/*ns-sources*` variables.

## Details of Namespace Loading

Using the above example, it can be helpful (especially when diagnosing failures loading namespaces) to understand how Joker constructs a pathname to a namespace to be loaded. This is done by `joker.core/lib-path__`, called by `(load ...)` and other core functions, directly or indirectly.

### The Current Namespace and Source File Path

When "in" a given namespace, Joker typically knows the pathname to the source code for the file defining the namespace. This is the "current file" (`joker.core/*file*`). It might start out as the file (such as a script) being run via the Joker command line. (When dropping into the REPL, or running code specified via `--eval`, `*file*` is nil.)

In the above example, the initial namespace is `ttt.core`, and its source file is (say) `/Users/somebody/mylibs/ttt/core.joke`.

### Referencing Another Namespace

Whether via `(require ...)`, `(load ...)`, `(use ...)`, or `(ns ...)`, the `lib-path__` procedure uses the current namespace and its source pathname to determine the location of the target namespace. (If there's no source pathname, such as when executing code entered via the REPL or `--eval`, the absolute pathname of the current directory is used, with a dummy filename (`user`) appended to it.)

It starts by "backing up" the source pathname, component by component, corresponding to the current namespace name (`joker.core/*ns*`). That is, for each component in `*ns*`, one basename is "stripped" from the current source pathname.

In the above example, this means that since `ttt.core` (the value of `*ns*` when executing the code in `core.joke`) has two components (`ttt` and `core`), two basenames are stripped from the source path. E.g. `/Users/somebody/mylibs/ttt/core.joke` has `core.joke` and then `ttt` stripped from it, yielding `/Users/somebody/mylibs`, which is treated as the "base path" for that namespace.

Then, when `core.joke` loads (via `(ns ... :require ...)`) `ttt.utils.a`, that target namespace is converted into the relative pathname `ttt/utils/a.joke` and appended to the base path (`/Users/somebody/mylibs`), determined above, yielding `/Users/somebody/mylibs/ttt/utils/a.joke`. This becomes the current pathname for `a.joke`.

While `a.joke` is being read and evaluated, `*ns*` soon becomes `ttt.utils.a` due to the `(ns ...)` invocation that starts the file.

So when `:require [ttt.utils.b ...]` is processed, `*ns*` has already been changed to `ttt.utils.a`, which causes the current base pathname to become `/Users/somebody/mylibs` again due to *three* basenames (`ttt/utils/a.joke`) being stripped from the current pathname (`*file*`). This is the same basename as was previously determined for `core.joke`.

Loading `ttt.utils.b` thus causes `/Users/somebody/mylibs/ttt/utils/b.joke` to be read.

### Diagnosing Problems

As one might conclude, from the above description, problems can arise when namespaces aren't consistently named with respect to the root resource (first file that defines the namespace) that defines them.

For example, if `a.joke` give its namespace name as `(ns ttt.utils.a.extra ...`, lookup of `ttt.utils.b` will (presumably) fail due to too many basenames (components) being stripped from the corresponding value of `*file*`:

```
<joker.core>:3536:13: Eval error: open /Users/somebody/ttt/utils/b.joke: no such file or directory
```

It might not be immediately obvious that the `mylibs/` component has been stripped from the above pathname.
