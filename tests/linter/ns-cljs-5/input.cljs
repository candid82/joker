(ns bar.core
 (:require [foo.core :as foo :include-macros true :exclude [t] :only g :refer [k] :rename {} :refer-macros []]))

(foo/bar)
