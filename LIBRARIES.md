<img src="https://user-images.githubusercontent.com/882970/48048842-a0224080-e151-11e8-8855-642cf5ef3fdd.png" width="117px"/>

# Organizing Libraries (Namespaces)

Joker is distributed with built-in namespaces. These include core namespaces (such as `joker.core` and `joker.math`) and Go-standard-library-wrapping namespaces (`joker.os`, `joker.http`, etc.). As they are built into the Joker executable itself, they need not be deployed separately from that executable. 

Currently, Joker does not include any additional Joker namespaces (library code) that is *not* built into the Joker executable. That makes the Joker executable "stand-alone" with respect to deployment.

However, Joker does provide support for deployment of additional namespaces via library code that is loaded during execution via `(ns ... :require ...)` and similar library-loading mechanisms. Though in its early stages (Joker itself being _v0.N_ at this writing), locally-developed libraries (such as those containing business logic) as well as third-party libraries are thereby supported.

This document provides a brief overview of these mechanisms and recommendations as to how to organize code for such namespaces.

## Default Behavior

Absent overriding behavior as defined by `joker.core/*classpath*` and `joker.core/*ns-sources*`, Joker normally relies on the local filesystem to locate source files for namespaces.

Generally, namespaces are converted to relative pathnames ("subpaths") by treating each component as a directory, except for the last, to which `.joke` is appended.

For example, `a.b.c` becomes `a/b/c.joke`.

Joker also keeps track of the pathname for the file currently being read and evaluated. Namespaces it seeks to load are searched relative to that pathname, taking into account the current namespace (`*ns*`).

So, if `/Users/somebody/mylibs/a/b/c.joke` is currently running as namespace `a.b.c`, and seeks to load the namespace `d.e`, the last three components (corresponding to `a`, `b`, and `c` of the current namespace) are removed from the current pathname, and the new subpath (`d/e.joke`) is appended, yielding `/Users/somebody/mylibs/d/e.joke`.

For more information on this default behavior, see [Library Loader Behavior](https://github.com/candid82/joker/blob/master/docs/misc/lib-loader.md).

## The \*classpath\* Variable

Though `lib-path__` determines a potential pathname for a library's root file, `*classpath*` determines whether and when to use that pathname.

The `joker.core/load-lib-from-path__` procedure, used to actually load library files, is called with the target namespace name and the pathname previously determined by `lib-path__`. It uses `*classpath*` to search for the `.joke` file representing the root source file for that namespace.

Each component of `*classpath*` (separated by colons on most OSes, semicolons on Windows) is consulted, in order, until the `.joke` file is found.

Only if the component is empty is the `lib-path__` pathname used (as described above). Since the default (empty) `*classpath*` consists of a single empty component, the `lib-path__` pathname is typically used with no further searching.

Non-empty components have the target namespace name, converted to pathname form and with `.joke` appended, appended to them, and the resulting paths are opened for reading. The first one that is successfully opened is used.

For example, if a component contains `/usr/lib/joker`, and the target namespace is `biz.logic`, the resulting pathname for the root file would be `/usr/lib/joker/biz/logic.joke`.

(Since `.` refers to the current working directory, a component of `.` would result in `./biz/logic.joke`. This is more of an artifact of the implementation than an expected usage.)

Thus, `*classpath*` provides a rudimentary mechanism for loading pre-existing (deployed) libraries. However, it has various shortcomings:

* It doesn't provide a delivery mechanism. Joker will try to load libraries from the specified location(s), but how those libraries get there is up to the user.

* It doesn't provide explicit version management. If many programs use the same shared library, it becomes tricky to update the library without breaking anything.

* It's external to programs' source code. One cannot tell which external dependencies will be loaded at runtime just by looking at the code.

## The \*ns-sources\* Variable

To address the limitations of `*classpath*`, `joker.core/*ns-sources*` (and the related helper function, `joker.core/ns-sources`) is provided.

`*ns-sources*` is intended as a next step in providing a best-in-class dependency management system, which should be:

* Simple, both conceptually and implementation-wise

* Explicit: it should be easy to tell which dependencies come from where just by looking at the source code

* Built-in: there should not be a separate tool or command to run to pull dependencies

### Form of *ns-sources*

Each element of the `*ns-sources*` vector is itself a two-element vector consisting of a:

* *key* that is a regular expression to be matched against the namespace name

* *value* that is a map specifying the location of the root source file for namespaces matching the *key*

`lib-path__` consults each element of `*ns-sources*`, in order, to see whether there's a match. If the namespace name matches a regex key, the corresponding value map's `:url` value is used to determine the pathname, as described below.

If no regex key matches the namespace name, `lib-path__` determines the path the usual way, as described above.

### The :url Value

Currently, two forms for the value of `:url` (in the map that is the value, or second element of the vector, matching the regexp key for the namespace) are supported:

* An HTTP URL, beginning with `http://` or `https://`

* Something else, which is treated as the base path for the namespace name

In both cases, the namespace name is converted into a relative path ("subpath") in the usual fashion, as described above. E.g. the namespace `a.b.c` becomes `a/b/c.joke`.

This subpath is appended to either the HTTP URL or to "something else". In the latter case, that becomes the pathname.

#### When :url Specifies HTTP Protocol

In the HTTP case, the subpath is appended to `$HOME/.jokerd/deps/<url>`, where `<url>` is the portion of the `:url` value following the `//` (and `$HOME` is the value of the `HOME` environment variable at the time of lookup). This pathname will be used for the locally cached version of the HTTP file.

For example, given a namespace of `a.b.c` and a `:url` of `https://example.com/joker/libs`, this becomes the pathname of the root source file of the namespace:

```
$HOME/.jokerd/deps/example.com/joker/libs/a/b/c.joke
```

(If necessary, the containing directory is created.)

If that file does not exist, Joker uses `net/http.Get()` to retrieve the target file based on the value for `:url`. If that value ends in `.joke`, it is used as-is; otherwise, the subpath (determined as described above) is appended to it.

In the above example, that means this URL is retrieved:

```
https://example.com/joker/libs/a/b/c.joke
```

Once retrieved, the contents are written to the (missing) file at the path shown above (in `$HOME/.jokerd`), so subsequent loads will use that cached file rather than repeatedly retrieving the contents via HTTP.

Regardless of whether the locally cached file initially exists, it is neither *read* nor *evaluated* in the Joker sense. That is, it is not parsed nor validated in any way; its contents are simply copied over, without analysis. Joker-style reading and evaluation is deferred until after `*classpath*` is consulted to determine whether the (cached) file is to be used at all.

### Examples

After running `(ns-sources {#"^mylibs[.]" {:url /Users/somebody/mylibs}})`, a `:require mylibs.awesome.code` would, since it matches the key in the outer map, try to load the root file from `/Users/somebody/mylibs/awesome/code.joke`.

### Current Limitations

* An HTTP failure is treated as a failure to load the namespace (library) even if `*classpath*` would match a local file. A workaround for this is to "touch" the cache file that would have been created, or (probably better yet) populate it with code that throws an error if it is actually invoked.

* There's currently no mechanism to determine whether the locally cached version of an HTTP resource is stale. This is related to the lack of versioning, modules, signatures, etc.

* One might expect `:reload` to ensure that the cached versions of relevant files are updated with the latest versions; that does not appear to be the case.

* The `ns-sources` function does very little validation of the arguments. This can lead to `*ns-sources*` being malformed, causing panics when loading libraries.

* There are no convenience functions to reset or remove entries from `*ns-sources*`.

* This document should be updated to explain how multiple files, defining a single namespace, should be handled. How would the root file load them? Does this interoperate with HTTP? Etc.

# Recommended Approaches to Library Organization

TBD.

# References

`(doc require)`

`(doc *file*)`

`(doc ns-sources)`

`(doc joker.core/*ns-sources*)`

`(doc joker.core/*classpath*)`

[Library Loader Behavior](https://github.com/candid82/joker/blob/master/docs/misc/lib-loader.md)

[Dependency Management](https://github.com/candid82/joker/issues/208)
