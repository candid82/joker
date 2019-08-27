<img src="https://user-images.githubusercontent.com/882970/48048842-a0224080-e151-11e8-8855-642cf5ef3fdd.png" width="117px"/>

[![CircleCI](https://circleci.com/gh/candid82/joker.svg?style=svg)](https://circleci.com/gh/candid82/joker)

Joker is a small Clojure interpreter and linter written in Go.

## Installation

On macOS, the easiest way to install Joker is via Homebrew:

```
brew install candid82/brew/joker
```

The same command can be used on Linux if you use [Linuxbrew](http://linuxbrew.sh/).

If you use Arch Linux, there is [AUR package](https://aur.archlinux.org/packages/joker/).

If you use [Nix](https://nixos.org/nix/), then you can install Joker with

```
nix-env -i joker
```

On other platforms (or if you prefer manual installation), download a [precompiled binary](https://github.com/candid82/joker/releases) for your platform and put it on your PATH.

You can also [build](#building) Joker from the source code.

## Usage

`joker` - launch REPL

`joker <filename>` - execute a script. Joker uses `.joke` filename extension. For example: `joker foo.joke`.

`joker -e <expression>` - execute an expression. For example: `joker -e '(println "Hello, world!")'`

`joker --lint <filename>` - lint a source file. See [Linter mode](#linter-mode) for more details.

`joker -` - execute a script on standard input (os.Stdin).

## Documentation

[Standard library reference](https://candid82.github.io/joker/)

[Joker slack channel](https://clojurians.slack.com/messages/C9VURUUNL/)

## Project goals

These are high level goals of the project that guide design and implementation decisions.

- Be suitable for scripting (lightweight, fast startup). This is something that Clojure is not good at and my personal itch I am trying to scratch.
- Be user friendly. Good error messages and stack traces are absolutely critical for programmer's happiness and productivity.
- Provide some tooling for Clojure and its dialects. Joker has [linter mode](#linter-mode) which can be used for linting Joker, Clojure and ClojureScript code. It catches some basic errors. For those who don't use Cursive, this is probably already better than the status quo. Joker can also be used for pretty printing EDN data structures (very basic algorithm at the moment). For example, the following command can be used to pretty print EDN data structure (read from stdin):

```
joker --hashmap-threshold -1 -e "(pprint (read))"
```

 There is [Sublime Text plugin](https://github.com/candid82/sublime-pretty-edn) that uses Joker for pretty printing EDN files. [Here](https://github.com/candid82/joker/releases/tag/v0.8.8) you can find the description of `--hashmap-threshold` parameter, if curious. Tooling is one of the primary Joker use cases for me, so I intend to improve and expand it.
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
  | Boolean    | bool                  |
  | Char       | rune                  |
  | Double     | float64               |
  | Int        | int                   |
  | Keyword    | n/a                   |
  | Nil        | n/a                   |
  | Ratio      | big.Rat               |
  | Regex      | regexp.Regexp         |
  | String     | string                |
  | Symbol     | n/a                   |
  | Time       | time.Time             |

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
1. The following features are not implemented: protocols, records, structmaps, chunked seqs, transients, tagged literals, unchecked arithmetics, primitive arrays, custom data readers, transducers, validators and watch functions for vars and atoms, hierarchies, sorted maps and sets.
1. Unrelated to the features listed above, the following function from clojure.core namespace are not currently implemented but will probably be implemented in some form in the future: `subseq`, `iterator-seq`, `reduced?`, `reduced`, `mix-collection-hash`, `definline`, `re-groups`, `hash-ordered-coll`, `enumeration-seq`, `compare-and-set!`, `rationalize`, `load-reader`, `find-keyword`, `comparator`, `resultset-seq`, `file-seq`, `sorted?`, `ensure-reduced`, `rsubseq`, `pr-on`, `seque`, `alter-var-root`, `hash-unordered-coll`, `re-matcher`, `unreduced`.
1. Built-in namespaces have `joker` prefix. The core namespace is called `joker.core`. Other built-in namespaces include `joker.string`, `joker.json`, `joker.os`, `joker.base64` etc. See [standard library reference](https://candid82.github.io/joker/) for details.
1. Miscellaneous:
  - `case` is just a syntactic sugar on top of `condp` and doesn't require options to be constants. It scans all the options sequentially.
  - `slurp` only takes one argument - a filename (string). No options are supported.
  - `ifn?` is called `callable?`
  - Map entry is represented as a two-element vector.

## Linter mode

To run Joker in linter mode pass `--lint --dialect <dialect>` flag, where `<dialect>` can be `clj`, `cljs`, `joker` or `edn`. If `--dialect <dialect>` is omitted, it will be set based on file extension. For example, `joker --lint foo.clj` will run linter for the file `foo.clj` using Clojure (as opposed to ClojureScript or Joker) dialect. `joker --lint --dialect cljs -` will run linter for standard input using ClojureScript dialect. Linter will read and parse all forms in the provided file (or read them from standard input) and output errors and warnings (if any) to standard output (for `edn` dialect it will only run read phase and won't parse anything). Let's say you have file `test.clj` with the following content:
```clojure
(let [a 1])
```
Executing the following command `joker --lint test.clj` will produce the following output:
```
test.clj:1:1: Parse warning: let form with empty body
```
The output format is as follows: `<filename>:<line>:<column>: <issue type>: <message>`, where `<issue type>` can be `Read error`, `Parse error`, `Parse warning` or `Exception`.

### Intergration with editors

- Emacs: [flycheck syntax checker](https://github.com/candid82/flycheck-joker)
- Sublime Text: [SublimeLinter plugin](https://github.com/candid82/SublimeLinter-contrib-joker)
- Atom: [linter-joker](https://atom.io/packages/linter-joker)
- Vim: [syntastic-joker](https://github.com/aclaimant/syntastic-joker), [ale](https://github.com/w0rp/ale)
- VSCode: [VSCode Linter Plugin (alpha)](https://github.com/martinklepsch/vscode-joker-clojure-linter)
- Kakoune: [clj-kakoune-joker](https://github.com/w33tmaricich/clj-kakoune-joker)

[Here](https://github.com/candid82/SublimeLinter-contrib-joker#reader-errors) are some examples of errors and warnings that the linter can output.

### Reducing false positives

Joker lints the code in one file at a time and doesn't try to resolve symbols from external namespaces. Because of that and since it's missing some Clojure(Script) features it doesn't always provide accurate linting. In general it tries to be unobtrusive and error on the side of false negatives rather than false positives. One common scenario that can lead to false positives is resolving symbols inside a macro. Consider the example below:

```clojure
(ns foo (:require [bar :refer [def-something]]))

(def-something baz ...)
```

Symbol `baz` is introduced inside `def-something` macro. The code is totally valid. However, the linter will output the following error: `Parse error: Unable to resolve symbol: baz`. This is because by default the linter assumes external vars (`bar/def-something` in this case) to hold functions, not macros. The good news is that you can tell Joker that `bar/def-something` is a macro and thus suppress the error message. To do that you need to add `bar/def-something` to the list of known macros in Joker configuration file. The configuration file is called `.joker` and should be in the same directory as the target file, or in its parent directory, or in its parent's parent directory etc up to the root directory. When reading from stdin Joker will look for a `.joker` file in the current working directory. The `--working-dir <path/to/file>` flag can be used to override the working directory that Joker starts looking in. Joker will also look for a `.joker` file in your home directory if it cannot find it in the above directories. The file should contain a single map with `:known-macros` key:

```clojure
{:known-macros [bar/def-something foo/another-macro ...]}
```

Please note that the symbols are namespace qualified and unquoted. Also, Joker knows about some commonly used macros (outside of `clojure.core` namespace) like `clojure.test/deftest` or `clojure.core.async/go-loop`, so you won't have to add those to your config file.

Joker also allows to specify symbols that are introduced by a macro:

```clojure
{:known-macros [[riemann.streams/where [service event]]]}
```

So each element in :known-macros vector can be either a symbol (as in the previous example) or a vector with two elements: macro's name and a list of symbols introduced by this macro. This allows to avoid symbol resolution warnings in macros that intern specific symbols implicitly.

Additionally, if you want Joker to ignore some unused namespaces (for example, if they are required for their side effects) you can add the `:ignored-unused-namespaces` key to your `.joker` file:

```clojure
{:ignored-unused-namespaces [foo.bar.baz]}
```

Sometimes your code may refer to a namespace that is not explicitly required in the same file. This is rarely needed, but if you face such situation you can add that namespace to `:known-namespaces` list to avoid "No namespace found" or "Unable to resolve symbol" warnings:

```clojure
{:known-namespaces [clojure.spec.gen.test]}
```

If your code uses tagged literals that Joker doesn't know about, add them to `:known-tags` list:

```clojure
{:known-tags [db/fn]}
```

If you use `:refer :all` Joker won't be able to properly resolve symbols because it doesn't know what vars are declared in the required namespace (i.e. `clojure.test`). There are generally three options here:

1. Refer specific symbols. For example: `[clojure.test :refer [deftest testing is are]]`. This is usually not too tedious, and you only need to do it once per file.
2. Use alias and qualified symbols:

 ```clojure
 (:require [clojure.test :as t])
 (t/deftest ...)
 ```

3. "Teach" Joker declarations from referred namespace. Joker executes the following files (if they exist) before linting your file: `.jokerd/linter.cljc` (for both Clojure and ClojureScript), `.jokerd/linter.clj` (Clojure only), `.jokerd/linter.cljs` (ClojureScript only). The rules for locating `.jokerd` directory are the same as for locating `.joker` file.

   *  :warning: Joker can be made aware of any additional declarations (like `deftest` and `is`) by providing them in `.jokerd/linter.clj[s|c]` files. However, this means Joker cannot check that the symbols really are declared in your namespace, so this feature should be used sparingly.
   * If you really want some symbols to be considered declared *in any namespace no matter what*, you can add `(in-ns 'joker.core)` to your `linter.clj[s|c]` and then declare those symbols.
    (see issues [52](https://github.com/candid82/joker/issues/52) and [50](https://github.com/candid82/joker/issues/50) for discussion).

I generally prefer first option for `clojure.test` namespace.

### Optional rules

Joker supports a few configurable linting rules. To turn them on or off set their values to `true` or `false` in `:rules` map in `.joker` file. For example:

```clojure
:rules {:if-without-else true
        :no-forms-threading false}
```

Below is the list of all configurable rules.

|          Rule          |                      Description                      | Default value |
|------------------------|-------------------------------------------------------|---------------|
| `if-without-else`      | warn on `if` without the `else` branch                | `false`       |
| `no-forms-threading`   | warn on threading macros with no forms, i.e. `(-> a)` | `true`        |
| `unused-as`            | warn on unused `:as` binding                          | `true`        |
| `unused-keys`          | warn on unused `:keys`, `:strs`, and `:syms` bindings | `true`        |
| `unused-fn-parameters` | warn on unused fn parameters                          | `false`       |
| `fn-with-empty-body`   | warn on fn form with empty body                       | `true`        |

Note that `unused binding` and `unused parameter` warnings are suppressed for names starting with underscore.

## Building

Joker requires Go v1.12 or later.
Below commands should get you up and running.

```
go get -d github.com/candid82/joker
cd $GOPATH/src/github.com/candid82/joker
./run.sh --version && go install
```

### Cross-platform Builds

After building the native version (to autogenerate appropriate files, "vet" the source code, etc.), set the appropriate environment variables and invoke `go build`. E.g.:

```
$ GOOS=linux GOARCH=arm GOARM=6 go build
```

(The `run.sh` script does not support cross-platform building.)

## Coding Guidelines

- Dashes (`-`) in namespaces are not converted to underscores (`_`) by Joker, so (unlike with Clojure) there's no need to name `.joke` files accordingly.
- Avoid `:refer :all` and the `use` function, as that reduces the effectiveness of linting.

## Contributors

(Generated by [Hall-Of-Fame](https://github.com/sourcerer-io/hall-of-fame))

[![](https://sourcerer.io/fame/candid82/candid82/joker/images/0)](https://sourcerer.io/fame/candid82/candid82/joker/links/0)[![](https://sourcerer.io/fame/candid82/candid82/joker/images/1)](https://sourcerer.io/fame/candid82/candid82/joker/links/1)[![](https://sourcerer.io/fame/candid82/candid82/joker/images/2)](https://sourcerer.io/fame/candid82/candid82/joker/links/2)[![](https://sourcerer.io/fame/candid82/candid82/joker/images/3)](https://sourcerer.io/fame/candid82/candid82/joker/links/3)[![](https://sourcerer.io/fame/candid82/candid82/joker/images/4)](https://sourcerer.io/fame/candid82/candid82/joker/links/4)[![](https://sourcerer.io/fame/candid82/candid82/joker/images/5)](https://sourcerer.io/fame/candid82/candid82/joker/links/5)[![](https://sourcerer.io/fame/candid82/candid82/joker/images/6)](https://sourcerer.io/fame/candid82/candid82/joker/links/6)[![](https://sourcerer.io/fame/candid82/candid82/joker/images/7)](https://sourcerer.io/fame/candid82/candid82/joker/links/7)

## License

```
Copyright (c) Roman Bataev. All rights reserved.
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


