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


