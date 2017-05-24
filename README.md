# Joker

[![CircleCI](https://circleci.com/gh/candid82/joker.svg?style=svg)](https://circleci.com/gh/candid82/joker)

Joker is a small Clojure interpreter and linter written in Go.

## Installation

On macOS, the easiest way to install Joker is via Homebrew:

```
brew install candid82/brew/joker
```

On other platforms (or if you prefer manual installation), download a [precompiled binary](https://github.com/candid82/joker/releases) for your platform and put it on your PATH.

You can also [build](#building) Joker from the source code.

## Usage

`joker` - launch REPL

`joker <filename>` - execute a script. Joker uses `.joke` filename extension. For example: `joker foo.joke`.

`joker --lint <filename>` - lint a source file. See [Linter mode](#linter-mode) for more details.

## Documentation

[Standard library reference](https://candid82.github.io/joker/)

## Project goals

These are high level goals of the project that guide design and implementation decisions.

- Be suitable for scripting (lightweight, fast startup). This is something that Clojure is not good at and my personal itch I am trying to scratch.
- Be user friendly. Good error messages and stack traces are absolutely critical for programmer's happiness and productivity.
- Provide some tooling for Clojure and its dialects. Joker has [linter mode](#linter-mode) which can be used for linting Joker, Clojure and ClojureScript code. It catches some basic errors. For those who don't use Cursive, this is probably already better than the status quo. Tooling is one of the primary Joker use cases for me, so I intend to improve and expand it.
- Be as close (syntactically and semantically) to Clojure as possible. Joker should truly be a dialect of Clojure, not a language inspired by Clojure. That said, there is a lot of Clojure features that Joker doesn't and will never have. Being close to Clojure only applies to features that Joker does have.

## Project non-goals

- Performance. If you need it, use Clojure. Joker is a naive implementation of an interpreter that evaluates unoptimized AST directly. I may be interested in doing some basic optimizations but this is definitely not a priority.
- Have all Clojure features. Some features are impossible to implement due to a different host language (Go vs Java), others I don't find that important for the use cases I have in mind for Joker. But generally Clojure is a pretty large language at this point and it is simply unfeasible to reach feature parity with it, even with naive implementation.

## Differences with Clojure

1. Primitive types are different due to a different host language and desire to simplify things. Scripting doesn't normally require all the integer and float types, for example. Here is a list of Joker's primitive types:

  | Joker type | Corresponding Go type |
  |------------|-----------------------|
  | BigFloat   | big.Float             |
  | BigInt     | big.Int               |
  | Bool       | bool                  |
  | Char       | rune                  |
  | Double     | float64               |
  | Int        | int                   |
  | Keyword    | n/a                   |
  | Nil        | n/a                   |
  | Ratio      | big.Rat               |
  | Regex      | regexp.Regexp         |
  | String     | string                |
  | Symbol     | n/a                   |

  Note that `Nil` is a type that has one value `nil`.

1. The set of persistent data structures is much smaller:

  | Joker type | Corresponding Clojure type |
  | ---------- | -------------------------- |
  | ArrayMap   | PersistentArrayMap         |
  | MapSet     | PersistentHashSet (or hypothetical PersistentArraySet, depending on which kind of underlying map is used) |
  | HashMap    | PersistentHashMap          |
  | List       | PersistentList             |
  | Vector     | PersistentVector           |

1. Joker doesn't have the same level of interoperability with the host language (Go) as Clojure does with Java or ClojureScript does with JavaScript. It doesn't have access to arbitrary Go types and functions. There is only a small fixed set of built-in types and interfaces. Dot notation for calling methods is not supported (as there are no methods). All Java/JVM specific functionality of Clojure is not implemented for obvious reasons.
1. Joker is single-threaded with no support for concurrency or parallelism. Therefore no refs, agents, futures, promises, locks, volatiles, transactions, `p*` functions that use multiple threads. Vars always have just one "root" binding.
1. The following features are not implemented: protocols, records, structmaps, multimethods, chunked seqs, transients, tagged literals, splicing reader conditionals (`#?@`), unchecked arithmetics, primitive arrays, custom data readers, transducers, validators and watch functions for vars and atoms, hierarchies, sorted maps and sets.
1. Unrelated to the features listed above, the following function from clojure.core namespace are not currently implemented but will probably be implemented in some form in the future: `subseq`, `iterator-seq`, `reduced?`, `reduced`, `mix-collection-hash`, `definline`, `re-groups`, `hash-ordered-coll`, `enumeration-seq`, `compare-and-set!`, `rationalize`, `clojure-version`, `load-reader`, `find-keyword`, `comparator`, `letfn`, `resultset-seq`, `line-seq`, `file-seq`, `sorted?`, `ensure-reduced`, `rsubseq`, `pr-on`, `seque`, `alter-var-root`, `hash-unordered-coll`, `re-matcher`, `unreduced`.
1. Built-in namespaces have `joker` prefix. The core namespace is called `joker.core`. Other namespaces (`joker.string`, `joker.json`, `joker.os`, `joker.base64`) are in their infancy.
1. Miscellaneous:
  1. `case` is just a syntactic sugar on top of `condp` and doesn't require options to be constants. It scans all the options sequentially.
  1. `refer-clojure` is not a thing. Use `(joker.core/refer 'joker.core)` instead if you really need to.
  1. `slurp` only takes one argument - a filename (string). No options are supported.
  1. `ifn?` is called `callable?`
  1. Map entry is represented as a two-element vector.

## Linter mode

To run Joker in linter mode pass `--lint<dialect>` flag, where `<dialect>` can be `clj`, `cljs`, `joker` or `edn`. If `<dialect>` is omitted, it will be set based on file extenstion. For example, `joker --lint foo.clj` will run linter for the file `foo.clj` using Clojure (as opposed to ClojureScript or Joker) dialect. `joker --lintcljs --` will run linter for standard input using ClojureScript dialect. Linter will read and parse all forms in the provided file (or read them from standard input) and output errors and warnings (if any) to standard output (for `edn` dialect it will only run read phase and won't parse anything). Let's say you have file `test.clj` with the following content:
```clojure
(let [a 1])
```
Executing the following command `joker --lint test.clj` will produce the following output:
```
test.clj:1:1: Parse warning: let form with empty body
```
The output format is as follows: `<filename>:<line>:<column> <issue type>: <message>`, where `<issue type` can be `Read error`, `Parse error`, `Parse warning` or `Exception`.

### Intergration with editors

- Emacs: [flycheck syntax checker](https://github.com/candid82/flycheck-joker)
- Sublime Text: [SublimeLinter plugin](https://github.com/candid82/SublimeLinter-contrib-joker)
- Atom: [linter-joker](https://atom.io/packages/linter-joker)

[Here](https://github.com/candid82/SublimeLinter-contrib-joker#reader-errors) are some examples of errors and warnings that the linter can output.

### Reducing false positives

Joker lints the code in one file at a time and doesn't try to resolve symbols from external namespaces. Because of that and since it's missing some Clojure(Script) features it doesn't always provide accurate linting. In general it tries to be unobtrusive and error on the side of false negatives rather than false positives. One common scenario that can lead to false positives is resolving symbols inside a macro. Consider the example below:

```clojure
(ns foo (:require [bar :refer [def-something]]))

(def-something baz ...)
```

Symbol `baz` is introduced inside `def-something` macro. The code it totally valid. However, the linter will output the following error: `Parse error: Unable to resolve symbol: baz`. This is because by default the linter assumes external vars (`bar/def-something` in this case) to hold functions, not macros. The good news is that you can tell Joker that `bar/def-something` is a macro and thus suppress the error message. To do that you need to add `bar/def-something` to the list of known macros in Joker configuration file. The configuration file is called `.joker` and should be in the same directory as the target file, or in its parent directory, or in its parent's parent directory etc up to the root directory. Joker will also look for `.joker` file in your home directory if it cannot find it in the above directories. The file should contain a single map with `:known-macros` key:

```clojure
{:known-macros [bar/def-something foo/another-macro ...]}
```

Please note that the symbols are namespace qualified and unquoted. Also, Joker knows about some commonly used macros (outside of `clojure.core` namespace) like `clojure.test/deftest` or `clojure.core.async/go-loop`, so you won't have to add those to your config file.

Additionally, if you want to ignore an unused namespace requirement you can add the `:ignored-unused-namespaces` key to your `.joker` file:

```clojure
{:ignored-unused-namespaces [foo.bar.baz]}
```

## Building

Joker's only dependency is [readline](https://github.com/chzyer/readline). However, part of Joker's source code is generated by Joker itself. So to build Joker, you have to have `joker` binary on your `PATH`.
Below commands should get you up and running. Ignore the error message about undefined `coreData` after you run `go get`: `coreData` will be generated by `go generate`.

```
go get github.com/candid82/joker
cd $GOPATH/src/github.com/candid82/joker
go generate ./...
go build
./joker
```

## License

```
Copyright (c) 2016 Roman Bataev. All rights reserved.
The use and distribution terms for this software are covered by the
Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
which can be found in the LICENSE file.
```

Joker contains parts of Clojure source code (from `clojure.core` namespace). Clojure is licensed as follows:

```
Copyright (c) Rich Hickey. All rights reserved.
The use and distribution terms for this software are covered by the
Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
which can be found in the file epl-v10.html at the root of this distribution.
By using this software in any fashion, you are agreeing to be bound by
the terms of this license.
You must not remove this notice, or any other, from this software.
```


