(ns foo
  (:refer-clojure :exclude [list]))

(defmacro bar [baz]
  `(identity ~baz))
