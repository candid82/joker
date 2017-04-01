(ns test
  (:require [test.ns1]
            [test.ns2 :as ns2]
            [test.ns3 :as ns3 :refer [f3]]
            [test.ns4 :as ns4 :refer [f4]]
            [test.ns5]
            [test.ns6 :as ns6]
            [test.ns7 :as n7]
            [test.ns8 :as n8]
            [test.ns9 :as n9]
            [test.n10])
  (:import my.JavaClass))

(f4)
(test.ns5/f5)
(ns6/f6)
(#'n7/f7)

(defmacro m8
  [& body]
  `(do
     (n8/f)
     ~@body))

::n9/k

`(test.n10/f10)
