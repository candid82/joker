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
  list list**)

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
  :doc "`assoc[iate]. When applied to a map, returns a new map of the
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

(def
  ^{:arglists '([obj])
  :doc "Returns the metadata of obj, returns nil if there is no metadata."
  :added "1.0"}
  meta meta*)

(def
  ^{:arglists '([obj m])
  :doc "Returns an object of the same type and value as obj, with
      map m as its metadata."
      :added "1.0"}
      with-meta with-meta*)

(def ^{:private true :dynamic true}
  assert-valid-fdecl (fn [fdecl]))

(def
  ^{:private true}
  sigs
  (fn [fdecl]
    (assert-valid-fdecl fdecl)
    (let [asig
          (fn [fdecl]
            (let [arglist (first fdecl)
                  ;elide implicit macro args
                  arglist (if (=* '&form (first arglist))
                            (subvec* arglist 2 (count* arglist))
                            arglist)
                  body (next fdecl)]
              (if (map? (first body))
                (if (next body)
                  (with-meta arglist (conj (if (meta arglist) (meta arglist) {}) (first body)))
                  arglist)
                arglist)))]
      (if (seq? (first fdecl))
        (loop [ret [] fdecls fdecl]
          (if fdecls
            (recur (conj ret (asig (first fdecls))) (next fdecls))
            (seq ret)))
        (list (asig fdecl))))))

(def
  ^{:arglists '([coll])
  :doc "Return the last item in coll, in linear time."
  :added "1.0"}
  last (fn last [s]
         (if (next s)
           (recur (next s))
           (first s))))

(def
  ^{:arglists '([coll])
  :doc "Return a seq of all but the last item in coll, in linear time."
  :added "1.0"}
  butlast (fn butlast [s]
            (loop [ret [] s s]
              (if (next s)
                (recur (conj ret (first s)) (next s))
                (seq ret)))))

(def

  ^{:doc "Same as (def name (fn [params* ] exprs*)) or (def
        name (fn ([params* ] exprs*)+)) with any doc-string or attrs added
        to the var metadata. prepost-map defines a map with optional keys
        :pre and :post that contain collections of pre or post conditions."
        :arglists '([name doc-string? attr-map? [params*] prepost-map? body]
                    [name doc-string? attr-map? ([params*] prepost-map? body)+ attr-map?])
        :added "1.0"}
        defn (fn defn [&form &env name & fdecl]
               ;; Note: Cannot delegate this check to def because of the call to (with-meta name ..)
               (if (instance? Symbol name)
                 nil
                 (throw (ex-info "First argument to defn must be a symbol" {:form name})))
               (let [m (if (string? (first fdecl))
                         {:doc (first fdecl)}
                         {})
                     fdecl (if (string? (first fdecl))
                             (next fdecl)
                             fdecl)
                     m (if (map? (first fdecl))
                         (conj m (first fdecl))
                         m)
                     fdecl (if (map? (first fdecl))
                             (next fdecl)
                             fdecl)
                     fdecl (if (vector? (first fdecl))
                             (list fdecl)
                             fdecl)
                     m (if (map? (last fdecl))
                         (conj m (last fdecl))
                         m)
                     fdecl (if (map? (last fdecl))
                             (butlast fdecl)
                             fdecl)
                     m (conj {:arglists (list 'quote (sigs fdecl))} m)
                     m (conj (if (meta name) (meta name) {}) m)]
                 (list 'def (with-meta name m)
                       (cons `fn fdecl) ))))

(set-macro* #'defn)

(defn cast
  "Throws an error if x is not of a type t, else returns x."
  {:added "1.0"}
  [^Type t x]
  (cast* t x))

(def
  ^{:arglists '([coll])
  :doc "Creates a new vector containing the contents of coll."
  :added "1.0"}
  vec vec*)

(defn vector
  "Creates a new vector containing the args."
  {:added "1.0"}
  [& args]
  (vec args))

(def
  ^{:arglists '([& keyvals])
  :doc "keyval => key val
      Returns a new hash map with supplied mappings.  If any keys are
      equal, they are handled as if by repeated uses of assoc."
      :added "1.0"}
      hash-map hash-map*)

(def
  ^{:arglists '([& keys])
  :doc "Returns a new hash set with supplied keys.  Any equal keys are
      handled as if by repeated uses of conj."
      :added "1.0"}
      hash-set hash-set*)

(defn nil?
  "Returns true if x is nil, false otherwise."
  {:tag Bool
  :added "1.0"}
  [x] (=* x nil))

(def

  ^{:doc "Like defn, but the resulting function name is declared as a
        macro and will be used as a macro by the compiler when it is
        called."
        :arglists '([name doc-string? attr-map? [params*] body]
                    [name doc-string? attr-map? ([params*] body)+ attr-map?])
        :added "1.0"}
        defmacro (fn [&form &env
                      name & args]
                   (let [prefix (loop [p (list name) args args]
                                  (let [f (first args)]
                                    (if (string? f)
                                      (recur (cons f p) (next args))
                                      (if (map? f)
                                        (recur (cons f p) (next args))
                                        p))))
                         fdecl (loop [fd args]
                                 (if (string? (first fd))
                                   (recur (next fd))
                                   (if (map? (first fd))
                                     (recur (next fd))
                                     fd)))
                         fdecl (if (vector? (first fdecl))
                                 (list fdecl)
                                 fdecl)
                         add-implicit-args (fn [fd]
                                             (let [args (first fd)]
                                               (cons (vec (cons '&form (cons '&env args))) (next fd))))
                         add-args (fn [acc ds]
                                    (if (nil? ds)
                                      acc
                                      (let [d (first ds)]
                                        (if (map? d)
                                          (conj acc d)
                                          (recur (conj acc (add-implicit-args d)) (next ds))))))
                         fdecl (seq (add-args [] fdecl))
                         decl (loop [p prefix d fdecl]
                                (if p
                                  (recur (next p) (cons (first p) d))
                                  d))]
                     (list 'do
                           (cons `defn decl)
                           (list 'set-macro* (list 'var name))
                           (list 'var name)))))

(set-macro* #'defmacro)

(defmacro when
  "Evaluates test. If logical true, evaluates body in an implicit do."
  {:added "1.0"}
  [test & body]
  (list 'if test (cons 'do body)))

(defmacro when-not
  "Evaluates test. If logical false, evaluates body in an implicit do."
  {:added "1.0"}
  [test & body]
  (list 'if test nil (cons 'do body)))

(defn false?
  "Returns true if x is the value false, false otherwise."
  {:tag Bool
  :added "1.0"}
  [x] (=* x false))

(defn true?
  "Returns true if x is the value true, false otherwise."
  {:tag Bool
  :added "1.0"}
  [x] (=* x true))

(defn not
  "Returns true if x is logical false, false otherwise."
  {:tag Bool
  :added "1.0"}
  [x] (if x false true))

(defn some?
  "Returns true if x is not nil, false otherwise."
  {:tag Bool
  :added "1.0"}
  [x] (not (nil? x)))

(def
  ^{:arglists '([& xs])
  :doc "With no args, returns the empty string. With one arg x, returns
      string representation of x. (str nil) returns the empty string. With more than
      one arg, returns the concatenation of the str values of the args."
      :added "1.0"
      :tag String}
      str str*)

(defn symbol?
  "Return true if x is a Symbol"
  {:added "1.0"}
  [x] (instance? Symbol x))

(defn keyword?
  "Return true if x is a Keyword"
  {:added "1.0"}
  [x] (instance? Keyword x))

(defn symbol
  "Returns a Symbol with the given namespace and name."
  {:tag Symbol
  :added "1.0"}
  ([name] (if (symbol? name) name (symbol* name)))
  ([ns name] (symbol* ns name)))

(defn gensym
  "Returns a new symbol with a unique name. If a prefix string is
  supplied, the name is prefix# where # is some unique number. If
  prefix is not supplied, the prefix is 'G__'."
  {:added "1.0"}
  ([] (gensym "G__"))
  ([prefix-string] (gensym* prefix-string)))

(defmacro cond
  "Takes a set of test/expr pairs. It evaluates each test one at a
  time.  If a test returns logical true, cond evaluates and returns
  the value of the corresponding expr and doesn't evaluate any of the
  other tests or exprs. (cond) returns nil."
  {:added "1.0"}
  [& clauses]
  (when clauses
    (list 'if (first clauses)
          (if (next clauses)
            (second clauses)
            (throw (ex-info "cond requires an even number of forms" {:form (first clauses)})))
          (cons 'gclojure.core/cond (next (next clauses))))))

(defn keyword
  "Returns a Keyword with the given namespace and name.  Do not use :
  in the keyword strings, it will be added automatically."
  {:tag Keyword
  :added "1.0"}
  ([name] (cond (keyword? name) name
            (symbol? name) (keyword* name)
            (string? name) (keyword* name)))
  ([ns name] (keyword* ns name)))

(defn spread
  {:private true}
  [arglist]
  (cond
    (nil? arglist) nil
    (nil? (next arglist)) (seq (first arglist))
    :else (cons (first arglist) (spread (next arglist)))))

(defn list*
  "Creates a new list containing the items prepended to the rest, the
  last of which will be treated as a sequence."
  {:added "1.0"}
  ([args] (seq args))
  ([a args] (cons a args))
  ([a b args] (cons a (cons b args)))
  ([a b c args] (cons a (cons b (cons c args))))
  ([a b c d & more]
   (cons a (cons b (cons c (cons d (spread more)))))))

(defn apply
  "Applies fn f to the argument list formed by prepending intervening arguments to args."
  {:added "1.0"}
  ([^Fn f args]
   (apply* f (seq args)))
  ([^Fn f x args]
   (apply* f (list* x args)))
  ([^Fn f x y args]
   (apply* f (list* x y args)))
  ([^Fn f x y z args]
   (apply* f (list* x y z args)))
  ([^Fn f a b c d & args]
   (apply* f (cons a (cons b (cons c (cons d (spread args))))))))

(defn vary-meta
  "Returns an object of the same type and value as obj, with
  (apply f (meta obj) args) as its metadata."
  {:added "1.0"}
  [obj f & args]
  (with-meta obj (apply f (meta obj) args)))

(defmacro lazy-seq
  "Takes a body of expressions that returns an ISeq or nil, and yields
  a Seqable object that will invoke the body only the first time seq
  is called, and will cache the result and return it on all subsequent
  seq calls. See also - realized?"
  {:added "1.0"}
  [& body]
  (list 'lazy-seq* (list* 'fn [] body)))

(defn chunked-seq? [s]
  ; Chunked sequences are not currently supported
  false)

(defn concat
  "Returns a lazy seq representing the concatenation of the elements in the supplied colls."
  {:added "1.0"}
  ([] (lazy-seq nil))
  ([x] (lazy-seq x))
  ([x y]
   (lazy-seq
    (let [s (seq x)]
      (if s
        (cons (first s) (concat (rest s) y))
        y))))
  ([x y & zs]
   (let [cat (fn cat [xys zs]
               (lazy-seq
                (let [xys (seq xys)]
                  (if xys
                    (cons (first xys) (cat (rest xys) zs))
                    (when zs
                      (cat (first zs) (next zs)))))))]
     (cat (concat x y) zs))))


(defmacro delay
  "Takes a body of expressions and yields a Delay object that will
  invoke the body only the first time it is forced (with force or deref/@), and
  will cache the result and return it on all subsequent force
  calls. See also - realized?"
  {:added "1.0"}
  [& body]
  (list 'delay* (list* 'fn [] body)))

(defn delay?
  "returns true if x is a Delay created with delay"
  {:added "1.0"}
  [x]
  (instance? Delay x))

(defn force
  "If x is a Delay, returns the (possibly cached) value of its expression, else returns x"
  {:added "1.0"}
  [x]
  (force* x))

(defmacro if-not
  "Evaluates test. If logical false, evaluates and returns then expr,
  otherwise else expr, if supplied, else nil."
  {:added "1.0"}
  ([test then] `(if-not ~test ~then nil))
  ([test then else]
   `(if (not ~test) ~then ~else)))

(defn identical?
  "Tests if 2 arguments are the same object"
  {:added "1.0"}
  [x y]
  (identical* x y))

(defn =
  "Equality. Returns true if x equals y, false if not. Works for nil, and compares
  numbers and collections in a type-independent manner.  Immutable data
  structures define = as a value, not an identity,
  comparison."
  {:added "1.0"}
  ([x] true)
  ([x y] (=* x y))
  ([x y & more]
   (if (=* x y)
     (if (next more)
       (recur y (first more) (next more))
       (=* y (first more)))
     false)))

(defn not=
  "Same as (not (= obj1 obj2))"
  {:tag Bool
  :added "1.0"}
  ([x] false)
  ([x y] (not (= x y)))
  ([x y & more]
   (not (apply = x y more))))

(defn compare
  "Comparator. Returns a negative number, zero, or a positive number
  when x is logically 'less than', 'equal to', or 'greater than'
  y. Works for nil, and compares numbers and collections in a type-independent manner. x
  must implement Comparable"
  {:added "1.0"}
  [x y] (compare* x y))

(defmacro and
  "Evaluates exprs one at a time, from left to right. If a form
  returns logical false (nil or false), and returns that value and
  doesn't evaluate any of the other expressions, otherwise it returns
  the value of the last expr. (and) returns true."
  {:added "1.0"}
  ([] true)
  ([x] x)
  ([x & next]
   `(let [and# ~x]
      (if and# (and ~@next) and#))))

(defmacro or
  "Evaluates exprs one at a time, from left to right. If a form
  returns a logical true value, or returns that value and doesn't
  evaluate any of the other expressions, otherwise it returns the
  value of the last expression. (or) returns nil."
  {:added "1.0"}
  ([] nil)
  ([x] x)
  ([x & next]
   `(let [or# ~x]
      (if or# or# (or ~@next)))))

(defn zero?
  "Returns true if num is zero, else false"
  {:added "1.0"}
  [x] (zero?* x))

(defn count
  "Returns the number of items in the collection. (count nil) returns
  0.  Also works on strings"
  {:added "1.0"}
  [coll] (count* coll))

(defn int
  "Coerce to int"
  {:added "1.0"}
  [x] (int* x))

(defn nth
  "Returns the value at the index. get returns nil if index out of
  bounds, nth throws an exception unless not-found is supplied.  nth
  also works, in O(n) time, for strings and sequences."
  {:added "1.0"}
  ([coll index] (nth* coll index))
  ([coll index not-found] (nth* coll index not-found)))

(defn <
  "Returns non-nil if nums are in monotonically increasing order,
  otherwise false."
  {:added "1.0"}
  ([x] true)
  ([x y] (<* x y))
  ([x y & more]
   (if (< x y)
     (if (next more)
       (recur y (first more) (next more))
       (< y (first more)))
     false)))

(defn inc'
  "Returns a number one greater than num. Supports arbitrary precision.
  See also: inc"
  {:added "1.0"}
  [x] (inc'* x))

(defn inc
  "Returns a number one greater than num. Does not auto-promote
  ints, will overflow. See also: inc'"
  {:added "1.0"}
  [x] (inc* x))

(defn ^:private
  reduce1
  ([f coll]
   (let [s (seq coll)]
     (if s
       (reduce1 f (first s) (next s))
       (f))))
  ([f val coll]
   (let [s (seq coll)]
     (if s
       (recur f (f val (first s)) (next s))
       val))))

(defn reverse
  "Returns a seq of the items in coll in reverse order. Not lazy."
  {:added "1.0"}
  [coll]
  (reduce1 conj () coll))

(defn +'
  "Returns the sum of nums. (+) returns 0. Supports arbitrary precision.
  See also: +"
  {:added "1.0"}
  ([] 0)
  ([x] (cast Number x))
  ([x y] (add'* x y))
  ([x y & more]
   (reduce1 +' (+' x y) more)))

(defn +
  "Returns the sum of nums. (+) returns 0. Does not auto-promote
  ints, will overflow. See also: +'"
  {:added "1.0"}
  ([] 0)
  ([x] (cast Number x))
  ([x y] (add* x y))
  ([x y & more]
   (reduce1 + (+ x y) more)))

(defn *'
  "Returns the product of nums. (*) returns 1. Supports arbitrary precision.
  See also: *"
  {:added "1.0"}
  ([] 1)
  ([x] (cast Number x))
  ([x y] (multiply'* x y))
  ([x y & more]
   (reduce1 *' (*' x y) more)))

(defn *
  "Returns the product of nums. (*) returns 1. Does not auto-promote
  ints, will overflow. See also: *'"
  {:added "1.0"}
  ([] 1)
  ([x] (cast Number x))
  ([x y] (multiply* x y))
  ([x y & more]
   (reduce1 * (* x y) more)))

(defn /
  "If no denominators are supplied, returns 1/numerator,
  else returns numerator divided by all of the denominators."
  {:added "1.0"}
  ([x] (/ 1 x))
  ([x y] (divide* x y))
  ([x y & more]
   (reduce1 / (/ x y) more)))

(defn -'
  "If no ys are supplied, returns the negation of x, else subtracts
  the ys from x and returns the result. Supports arbitrary precision.
  See also: -"
  {:added "1.0"}
  ([x] (subtract'* x))
  ([x y] (subtract'* x y))
  ([x y & more]
   (reduce1 -' (-' x y) more)))

(defn -
  "If no ys are supplied, returns the negation of x, else subtracts
  the ys from x and returns the result. Does not auto-promote
  ints, will overflow. See also: -'"
  {:added "1.0"}
  ([x] (subtract* x))
  ([x y] (subtract* x y))
  ([x y & more]
   (reduce1 - (- x y) more)))

(defn <=
  "Returns non-nil if nums are in monotonically non-decreasing order,
  otherwise false."
  {:added "1.0"}
  ([x] true)
  ([x y] (<=* x y))
  ([x y & more]
   (if (<= x y)
     (if (next more)
       (recur y (first more) (next more))
       (<= y (first more)))
     false)))

(defn >
  "Returns non-nil if nums are in monotonically decreasing order,
  otherwise false."
  {:added "1.0"}
  ([x] true)
  ([x y] (>* x y))
  ([x y & more]
   (if (> x y)
     (if (next more)
       (recur y (first more) (next more))
       (> y (first more)))
     false)))

(defn >=
  "Returns non-nil if nums are in monotonically non-increasing order,
  otherwise false."
  {:added "1.0"}
  ([x] true)
  ([x y] (>=* x y))
  ([x y & more]
   (if (>= x y)
     (if (next more)
       (recur y (first more) (next more))
       (>= y (first more)))
     false)))

(defn ==
  "Returns non-nil if nums all have the equivalent
  value (type-independent), otherwise false"
  {:added "1.0"}
  ([x] true)
  ([x y] (==* x y))
  ([x y & more]
   (if (== x y)
     (if (next more)
       (recur y (first more) (next more))
       (== y (first more)))
     false)))

(defn max
  "Returns the greatest of the nums."
  {:added "1.0"}
  ([x] x)
  ([x y] (max* x y))
  ([x y & more]
   (reduce1 max (max x y) more)))

(defn min
  "Returns the least of the nums."
  {:added "1.0"}
  ([x] x)
  ([x y] (min* x y))
  ([x y & more]
   (reduce1 min (min x y) more)))

(defn dec'
  "Returns a number one less than num. Supports arbitrary precision.
  See also: dec"
  {:added "1.0"}
  [x] (dec'* x))

(defn dec
  "Returns a number one less than num. Does not auto-promote
  ints, will overflow. See also: dec'"
  {:added "1.0"}
  [x] (dec* x))

(defn pos?
  "Returns true if num is greater than zero, else false"
  {:added "1.0"}
  [x] (pos* x))

(defn neg?
  "Returns true if num is less than zero, else false"
  {:added "1.0"}
  [x] (neg* x))

(defn quot
  "quot[ient] of dividing numerator by denominator."
  {:added "1.0"}
  [num div]
  (quot* num div))

(defn rem
  "remainder of dividing numerator by denominator."
  {:added "1.0"}
  [num div]
  (rem* num div))

(defn bit-not
  "Bitwise complement"
  {:added "1.0"}
  [x] (bit-not* x))

(defn bit-and
  "Bitwise and"
  {:added "1.0"}
  ([x y] (bit-and* x y))
  ([x y & more]
   (reduce1 bit-and (bit-and x y) more)))

(defn bit-or
  "Bitwise or"
  {:added "1.0"}
  ([x y] (bit-or* x y))
  ([x y & more]
   (reduce1 bit-or (bit-or x y) more)))

(defn bit-xor
  "Bitwise exclusive or"
  {:added "1.0"}
  ([x y] (bit-xor* x y))
  ([x y & more]
   (reduce1 bit-xor (bit-xor x y) more)))

(defn bit-and-not
  "Bitwise and with complement"
  {:added "1.0"}
  ([x y] (bit-and-not* x y))
  ([x y & more]
   (reduce1 bit-and-not (bit-and-not x y) more)))

(defn bit-clear
  "Clear bit at index n"
  {:added "1.0"}
  [x n] (bit-clear* x n))

(defn bit-set
  "Set bit at index n"
  {:added "1.0"}
  [x n] (bit-set* x n))

(defn bit-flip
  "Flip bit at index n"
  {:added "1.0"}
  [x n] (bit-flip* x n))

(defn bit-test
  "Test bit at index n"
  {:added "1.0"}
  [x n] (bit-test* x n))

(defn bit-shift-left
  "Bitwise shift left"
  {:added "1.0"}
  [x n] (bit-shift-left* x n))

(defn bit-shift-right
  "Bitwise shift right"
  {:added "1.0"}
  [x n] (bit-shift-right* x n))

(defn unsigned-bit-shift-right
  "Bitwise shift right, without sign-extension."
  {:added "1.0"}
  [x n] (unsigned-bit-shift-right* x n))

(defn integer?
  "Returns true if n is an integer"
  {:static true}
  [n]
  (or (instance? Int n)
      (instance? BigInt n)))

(defn even?
  "Returns true if n is even, throws an exception if n is not an integer"
  {:added "1.0"}
  [n]
  (if (integer? n)
    (zero? (bit-and (int* n) 1))
    (throw (ex-info (str "Argument must be an integer: " n) {}))))

(defn odd?
  "Returns true if n is odd, throws an exception if n is not an integer"
  {:added "1.0"}
  [n] (not (even? n)))

(defn complement
  "Takes a fn f and returns a fn that takes the same arguments as f,
  has the same effects, if any, and returns the opposite truth value."
  {:added "1.0"}
  [f]
  (fn
    ([] (not (f)))
    ([x] (not (f x)))
    ([x y] (not (f x y)))
    ([x y & zs] (not (apply f x y zs)))))

(defn constantly
  "Returns a function that takes any number of arguments and returns x."
  {:added "1.0"}
  [x]
  (fn [& args] x))

(defn identity
  "Returns its argument."
  {:added "1.0"}
  [x] x)

(defn peek
  "For a list, same as first, for a vector, same as, but much
  more efficient than, last. If the collection is empty, returns nil."
  {:added "1.0"}
  [coll] (peek* coll))

(defn pop
  "For a list, returns a new list without the first
  item, for a vector, returns a new vector without the last item. If
  the collection is empty, throws an exception.  Note - not the same
  as next/butlast."
  {:added "1.0"}
  [coll] (pop* coll))

(defn contains?
  "Returns true if key is present in the given collection, otherwise
  returns false.  Note that for numerically indexed collections like
  vectors, this tests if the numeric key is within the
  range of indexes. 'contains?' operates constant or logarithmic time;
  it will not perform a linear search for a value.  See also 'some'."
  {:added "1.0"}
  [coll key] (contains?* coll key))

(defn get
  "Returns the value mapped to key, not-found or nil if key not present."
  {:added "1.0"}
  ([map key]
   (get* map key))
  ([map key not-found]
   (get* map key not-found)))

(defn dissoc
  "dissoc[iate]. Returns a new map of the same (hashed/sorted) type,
  that does not contain a mapping for key(s)."
  {:added "1.0"}
  ([map] map)
  ([map key]
   (dissoc* map key))
  ([map key & ks]
   (let [ret (dissoc* map key)]
     (if ks
       (recur ret (first ks) (next ks))
       ret))))

(defn disj
  "disj[oin]. Returns a new set of the same (hashed/sorted) type, that
  does not contain key(s)."
  {:added "1.0"}
  ([set] set)
  ([set key]
   (disj* set key))
  ([set key & ks]
   (when set
     (let [ret (disj* set key)]
       (if ks
         (recur ret (first ks) (next ks))
         ret)))))

(defn find
  "Returns the map entry for key, or nil if key not present."
  {:added "1.0"}
  [map key] (find* map key))

(defn select-keys
  "Returns a map containing only those entries in map whose key is in keys"
  {:added "1.0"}
  [map keyseq]
  (loop [ret {} keys (seq keyseq)]
    (if keys
      (let [entry (find* map (first keys))]
        (recur
         (if entry
           (conj ret entry)
           ret)
         (next keys)))
      (with-meta ret (meta map)))))

(defn keys
  "Returns a sequence of the map's keys, in the same order as (seq map)."
  {:added "1.0"}
  [map] (keys* map))

(defn vals
  "Returns a sequence of the map's values, in the same order as (seq map)."
  {:added "1.0"}
  [map] (vals* map))

(defn key
  "Returns the key of the map entry."
  {:added "1.0"}
  [e]
  (first e))

(defn val
  "Returns the value in the map entry."
  {:added "1.0"}
  [e]
  (second e))

(defn rseq
  "Returns, in constant time, a seq of the items in rev (which
  can be a vector or sorted-map), in reverse order. If rev is empty returns nil."
  {:added "1.0"}
  [^Reversible rev]
  (rseq* rev))

(defn name
  "Returns the name String of a string, symbol or keyword."
  {:tag String
  :added "1.0"}
  [x]
  (if (string? x) x (name* x)))

(defn namespace
  "Returns the namespace String of a symbol or keyword, or nil if not present."
  {:tag String
  :added "1.0"}
  [^Named x]
  (namespace* x))

(defmacro ->
  "Threads the expr through the forms. Inserts x as the
  second item in the first form, making a list of it if it is not a
  list already. If there are more forms, inserts the first form as the
  second item in second form, etc."
  {:added "1.0"}
  [x & forms]
  (loop [x x, forms forms]
    (if forms
      (let [form (first forms)
            threaded (if (seq? form)
                       (with-meta `(~(first form) ~x ~@(next form)) (meta form))
                       (list form x))]
        (recur threaded (next forms)))
      x)))

(defmacro ->>
  "Threads the expr through the forms. Inserts x as the
  last item in the first form, making a list of it if it is not a
  list already. If there are more forms, inserts the first form as the
  last item in second form, etc."
  {:added "1.0"}
  [x & forms]
  (loop [x x, forms forms]
    (if forms
      (let [form (first forms)
            threaded (if (seq? form)
                       (with-meta `(~(first form) ~@(next form)  ~x) (meta form))
                       (list form x))]
        (recur threaded (next forms)))
      x)))

(defmacro ^{:private true} assert-args
  [& pairs]
  `(do (when-not ~(first pairs)
         (throw (ex-info
                 (str (first ~'&form) " requires " ~(second pairs))
                 {:form ~'&form})))
     ~(let [more (nnext pairs)]
        (when more
          (list* `assert-args more)))))

(defmacro if-let
  "bindings => binding-form test

  If test is true, evaluates then with binding-form bound to the value of
  test, if not, yields else"
  {:added "1.0"}
  ([bindings then]
   `(if-let ~bindings ~then nil))
  ([bindings then else & oldform]
   (assert-args
    (vector? bindings) "a vector for its binding"
    (nil? oldform) "1 or 2 forms after binding vector"
    (= 2 (count bindings)) "exactly 2 forms in binding vector")
   (let [form (bindings 0) tst (bindings 1)]
     `(let [temp# ~tst]
        (if temp#
          (let [~form temp#]
            ~then)
          ~else)))))

(defmacro when-let
  "bindings => binding-form test

  When test is true, evaluates body with binding-form bound to the value of test"
  {:added "1.0"}
  [bindings & body]
  (assert-args
   (vector? bindings) "a vector for its binding"
   (= 2 (count bindings)) "exactly 2 forms in binding vector")
  (let [form (bindings 0) tst (bindings 1)]
    `(let [temp# ~tst]
       (when temp#
         (let [~form temp#]
           ~@body)))))

(defmacro if-some
  "bindings => binding-form test

  If test is not nil, evaluates then with binding-form bound to the
  value of test, if not, yields else"
  {:added "1.0"}
  ([bindings then]
   `(if-some ~bindings ~then nil))
  ([bindings then else & oldform]
   (assert-args
    (vector? bindings) "a vector for its binding"
    (nil? oldform) "1 or 2 forms after binding vector"
    (= 2 (count bindings)) "exactly 2 forms in binding vector")
   (let [form (bindings 0) tst (bindings 1)]
     `(let [temp# ~tst]
        (if (nil? temp#)
          ~else
          (let [~form temp#]
            ~then))))))

(defmacro when-some
  "bindings => binding-form test

  When test is not nil, evaluates body with binding-form bound to the
  value of test"
  {:added "1.0"}
  [bindings & body]
  (assert-args
   (vector? bindings) "a vector for its binding"
   (= 2 (count bindings)) "exactly 2 forms in binding vector")
  (let [form (bindings 0) tst (bindings 1)]
    `(let [temp# ~tst]
       (if (nil? temp#)
         nil
         (let [~form temp#]
           ~@body)))))

(defn find-var
  "Returns the global var named by the namespace-qualified symbol, or
  nil if no var with that name."
  {:added "1.0"}
  [sym] (find-var* sym))

(defn comp
  "Takes a set of functions and returns a fn that is the composition
  of those fns.  The returned fn takes a variable number of args,
  applies the rightmost of fns to the args, the next
  fn (right-to-left) to the result, etc."
  {:added "1.0"}
  ([] identity)
  ([f] f)
  ([f g]
   (fn
     ([] (f (g)))
     ([x] (f (g x)))
     ([x y] (f (g x y)))
     ([x y z] (f (g x y z)))
     ([x y z & args] (f (apply g x y z args)))))
  ([f g h]
   (fn
     ([] (f (g (h))))
     ([x] (f (g (h x))))
     ([x y] (f (g (h x y))))
     ([x y z] (f (g (h x y z))))
     ([x y z & args] (f (g (apply h x y z args))))))
  ([f1 f2 f3 & fs]
   (let [fs (reverse (list* f1 f2 f3 fs))]
     (fn [& args]
       (loop [ret (apply (first fs) args) fs (next fs)]
         (if fs
           (recur ((first fs) ret) (next fs))
           ret))))))

(defn juxt
  "Takes a set of functions and returns a fn that is the juxtaposition
  of those fns.  The returned fn takes a variable number of args, and
  returns a vector containing the result of applying each fn to the
  args (left-to-right).
  ((juxt a b c) x) => [(a x) (b x) (c x)]"
  {:added "1.0"}
  ([f]
   (fn
     ([] [(f)])
     ([x] [(f x)])
     ([x y] [(f x y)])
     ([x y z] [(f x y z)])
     ([x y z & args] [(apply f x y z args)])))
  ([f g]
   (fn
     ([] [(f) (g)])
     ([x] [(f x) (g x)])
     ([x y] [(f x y) (g x y)])
     ([x y z] [(f x y z) (g x y z)])
     ([x y z & args] [(apply f x y z args) (apply g x y z args)])))
  ([f g h]
   (fn
     ([] [(f) (g) (h)])
     ([x] [(f x) (g x) (h x)])
     ([x y] [(f x y) (g x y) (h x y)])
     ([x y z] [(f x y z) (g x y z) (h x y z)])
     ([x y z & args] [(apply f x y z args) (apply g x y z args) (apply h x y z args)])))
  ([f g h & fs]
   (let [fs (list* f g h fs)]
     (fn
       ([] (reduce1 #(conj %1 (%2)) [] fs))
       ([x] (reduce1 #(conj %1 (%2 x)) [] fs))
       ([x y] (reduce1 #(conj %1 (%2 x y)) [] fs))
       ([x y z] (reduce1 #(conj %1 (%2 x y z)) [] fs))
       ([x y z & args] (reduce1 #(conj %1 (apply %2 x y z args)) [] fs))))))

(defn partial
  "Takes a function f and fewer than the normal arguments to f, and
  returns a fn that takes a variable number of additional args. When
  called, the returned function calls f with args + additional args."
  {:added "1.0"}
  ([f] f)
  ([f arg1]
   (fn [& args] (apply f arg1 args)))
  ([f arg1 arg2]
   (fn [& args] (apply f arg1 arg2 args)))
  ([f arg1 arg2 arg3]
   (fn [& args] (apply f arg1 arg2 arg3 args)))
  ([f arg1 arg2 arg3 & more]
   (fn [& args] (apply f arg1 arg2 arg3 (concat more args)))))

(defn sequence
  "Coerces coll to a (possibly empty) sequence, if it is not already
  one. Will not force a lazy seq. (sequence nil) yields ()"
  {:added "1.0"}
  [coll]
  (if (seq? coll)
    coll
    (or (seq coll) ())))

(defn every?
  "Returns true if (pred x) is logical true for every x in coll, else
  false."
  {:tag Bool
  :added "1.0"}
  [pred coll]
  (cond
    (nil? (seq coll)) true
    (pred (first coll)) (recur pred (next coll))
    :else false))

(def
  ^{:tag Bool
    :doc "Returns false if (pred x) is logical true for every x in
         coll, else true."
         :arglists '([pred coll])
         :added "1.0"}
  not-every? (comp not every?))

(defn some
  "Returns the first logical true value of (pred x) for any x in coll,
  else nil.  One common idiom is to use a set as pred, for example
  this will return :fred if :fred is in the sequence, otherwise nil:
  (some #{:fred} coll)"
  {:added "1.0"}
  [pred coll]
  (when (seq coll)
    (or (pred (first coll)) (recur pred (next coll)))))

(def
  ^{:tag Bool
    :doc "Returns false if (pred x) is logical true for any x in coll,
         else true."
         :arglists '([pred coll])
         :added "1.0"}
  not-any? (comp not some))

;will be redefed later with arg checks
(defmacro dotimes
  "bindings => name n

  Repeatedly executes body (presumably for side-effects) with name
  bound to integers from 0 through n-1."
  {:added "1.0"}
  [bindings & body]
  (let [i (first bindings)
        n (second bindings)]
    `(let [n# (int* ~n)]
       (loop [~i 0]
         (when (< ~i n#)
           ~@body
           (recur (inc ~i)))))))

(defn map
  "Returns a lazy sequence consisting of the result of applying f to the
  set of first items of each coll, followed by applying f to the set
  of second items in each coll, until any one of the colls is
  exhausted.  Any remaining items in other colls are ignored. Function
  f should accept number-of-colls arguments."
  {:added "1.0"}
  ([f coll]
   (lazy-seq
    (when-let [s (seq coll)]
      (cons (f (first s)) (map f (rest s))))))
  ([f c1 c2]
   (lazy-seq
    (let [s1 (seq c1) s2 (seq c2)]
      (when (and s1 s2)
        (cons (f (first s1) (first s2))
              (map f (rest s1) (rest s2)))))))
  ([f c1 c2 c3]
   (lazy-seq
    (let [s1 (seq c1) s2 (seq c2) s3 (seq c3)]
      (when (and  s1 s2 s3)
        (cons (f (first s1) (first s2) (first s3))
              (map f (rest s1) (rest s2) (rest s3)))))))
  ([f c1 c2 c3 & colls]
   (let [step (fn step [cs]
                (lazy-seq
                 (let [ss (map seq cs)]
                   (when (every? identity ss)
                     (cons (map first ss) (step (map rest ss)))))))]
     (map #(apply f %) (step (conj colls c3 c2 c1))))))

(defn mapcat
  "Returns the result of applying concat to the result of applying map
  to f and colls.  Thus function f should return a collection."
  {:added "1.0"}
  [f & colls]
  (apply concat (apply map f colls)))

(defn filter
  "Returns a lazy sequence of the items in coll for which
  (pred item) returns true. pred must be free of side-effects."
  {:added "1.0"}
  ([pred coll]
   (lazy-seq
    (when-let [s (seq coll)]
      (let [f (first s) r (rest s)]
        (if (pred f)
          (cons f (filter pred r))
          (filter pred r)))))))
