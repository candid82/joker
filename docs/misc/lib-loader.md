Lib loader assumes that namespaces' names correspond to their location in the filesystem. Specifically, last part of the namespace (after the last dot) should match the file name (without the `.joke` extension) and preceding parts correspond to the path to the file (with dots separating directories).
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

Then you can run `core.joke` and get the following (expected) output:

```
I am A
I am B
```

This mechanism is similar to how class loading works on JVM (and how namespace loading works in Clojure). Please note that current working directory doesn't come into play here: `lib-path__` resolves libraries' paths relative to the file currently being executed.
