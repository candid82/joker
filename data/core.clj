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
  also works for strings, and, in O(n) time, for sequences."
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
