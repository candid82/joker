; This code is a modified version of original Clojure's implementation
; (https://github.com/clojure/clojure/blob/master/src/clj/clojure/core.clj)
; which includes the following notice:

;   Copyright (c) Rich Hickey. All rights reserved.
;   The use and distribution terms for this software are covered by the
;   Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
;   which can be found in the file epl-v10.html at the root of this distribution.
;   By using this software in any fashion, you are agreeing to be bound by
;   the terms of this license.
;   You must not remove this notice, or any other, from this software.


(def unquote)
(def unquote-splicing)

(def
 ^{:arglists '([& items])
   :doc "Creates a new list containing the items."
   :added "1.0"}
  list list*)

(def
 ^{:arglists '([x seq])
    :doc "Returns a new seq where x is the first element and seq is
    the rest."
   :added "1.0"}
 cons cons*)

(def
 ^{:arglists '([coll])
   :doc "Returns the first item in the collection. Calls seq on its
    argument. If coll is nil, returns nil."
   :added "1.0"}
 first first*)

(def
 ^{:arglists '([coll])
   :doc "Returns a seq of the items after the first. Calls seq on its
  argument.  If there are no more items, returns nil."
   :added "1.0"}
 next next*)

(def
 ^{:arglists '([coll])
   :doc "Returns a possibly empty seq of the items after the first. Calls seq on its
  argument."
   :added "1.0"}
 rest rest*)

(def
 ^{:arglists '([coll x] [coll x & xs])
   :doc "conj[oin]. Returns a new collection with the xs
    'added'. (conj nil item) returns (item).  The 'addition' may
    happen at different 'places' depending on the concrete type."
   :added "1.0"}
 conj (fn conj
        ([coll x] (conj* coll x))
        ([coll x & xs]
         (if xs
           (recur (conj* coll x) (first xs) (next xs))
           (conj* coll x)))))

(def
 ^{:doc "Same as (first (next x))"
   :arglists '([x])
   :added "1.0"}
 second (fn second [x] (first (next x))))

(def
 ^{:doc "Same as (first (first x))"
   :arglists '([x])
   :added "1.0"}
 ffirst (fn ffirst [x] (first (first x))))

(def
 ^{:doc "Same as (next (first x))"
   :arglists '([x])
   :added "1.0"}
 nfirst (fn nfirst [x] (next (first x))))

(def
 ^{:doc "Same as (first (next x))"
   :arglists '([x])
   :added "1.0"}
 fnext (fn fnext [x] (first (next x))))

(def
 ^{:doc "Same as (next (next x))"
   :arglists '([x])
   :added "1.0"}
 nnext (fn nnext [x] (next (next x))))

(def
 ^{:arglists '([coll])
   :doc "Returns a seq on the collection. If the collection is
    empty, returns nil.  (seq nil) returns nil."
   :added "1.0"}
 seq seq*)

(def
 ^{:arglists '([c x])
   :doc "Evaluates x and tests if it is an instance of type
    c. Returns true or false"
   :added "1.0"}
 instance? instance?*)

(def
 ^{:arglists '([x])
   :doc "Returns true if x is a sequence"
   :added "1.0"}
 seq? (fn seq? [x] (instance? Seq x)))

(def
 ^{:arglists '([x])
   :doc "Returns true if x is a Char"
   :added "1.0"}
 char? (fn char? [x] (instance? Char x)))

(def
 ^{:arglists '([x])
   :doc "Returns true if x is a String"
   :added "1.0"}
 string? (fn string? [x] (instance? String x)))

(def
 ^{:arglists '([x])
   :doc "Returns true if x is a map"
   :added "1.0"}
 map? (fn map? [x] (instance? ArrayMap x)))

(def
 ^{:arglists '([x])
   :doc "Returns true if x is a vector"
   :added "1.0"}
 vector? (fn vector? [x] (instance? Vector x)))

(def
 ^{:arglists '([map key val] [map key val & kvs])
   :doc "assoc[iate]. When applied to a map, returns a new map of the
    same (hashed/sorted) type, that contains the mapping of key(s) to
    val(s). When applied to a vector, returns a new vector that
    contains val at index. Note - index must be <= (count vector)."
   :added "1.0"}
 assoc
 (fn assoc
   ([map key val] (assoc* map key val))
   ([map key val & kvs]
    (let [ret (assoc* map key val)]
      (if kvs
        (if (next kvs)
          (recur ret (first kvs) (second kvs) (nnext kvs))
          (throw (ex-info "assoc expects even number of arguments after map/vector, found odd number" {})))
        ret)))))
