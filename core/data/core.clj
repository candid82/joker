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

;during bootstrap we don't have destructuring let, loop or fn, will redefine later
(def
  ^{:added "1.0"}
  let (fn* let [&form &env & decl] (cons 'let* decl)))

(set-macro* #'let)

(def
  ^{:added "1.0"}
  loop (fn* loop [&form &env & decl] (cons 'loop* decl)))

(set-macro* #'loop)

(def
  ^{:added "1.0"}
  fn (fn* fn [&form &env & decl]
          (with-meta*
            (cons 'fn* decl)
            (meta* &form))))

(set-macro* #'fn)

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
  ^{:arglists '([msg map] [msg map cause])
    :doc "Create an instance of ExInfo, an Error that carries a map of additional data."
    :added "1.0"}
  ex-info ex-info*)

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
                 (cons `fn fdecl)))))

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
          (cons 'joker.core/cond (next (next clauses))))))

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
  (loop [x x forms forms]
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

(defn reduce-kv
  "Reduces an associative collection. f should be a function of 3
  arguments. Returns the result of applying f to init, the first key
  and the first value in coll, then applying f to that result and the
  2nd key and value, etc. If coll contains no entries, returns init
  and f is not called. Note that reduce-kv is supported on vectors,
  where the keys will be the ordinals."
  {:added "1.0"}
  ([f init coll]
   (reduce1 (fn [ret kv] (f ret (first kv) (second kv))) init coll)))

(defn var-get
  "Gets the value in the var object"
  {:added "1.0"}
  [^Var x] (var-get* x))

(defn var-set
  "Sets the value in the var object to val."
  {:added "1.0"}
  [^Var x val] (var-set* x val))

(defn replace-bindings*
  [binding-map]
  (reduce-kv (fn [res k v]
               (let [c (var-get k)]
                 (var-set k v)
                 (assoc res k c)))
             {}
             binding-map))

(defn with-bindings*
  "Takes a map of Var/value pairs. Sets the vars to the corresponding values.
  Then calls f with the supplied arguments. Resets the vars back to the original
  values after f returned. Returns whatever f returns."
  {:added "1.0"}
  [binding-map f & args]
  (let [existing-bindings (replace-bindings* binding-map)]
    (try
      (apply f args)
      (finally
        (replace-bindings* existing-bindings)))))

(defmacro with-bindings
  "Takes a map of Var/value pairs. Sets the vars to the corresponding values.
  Then executes body. Resets the vars back to the original
  values after body was evaluated. Returns the value of body."
  {:added "1.0"}
  [binding-map & body]
  `(with-bindings* ~binding-map (fn [] ~@body)))

(defmacro binding
  "binding => var-symbol init-expr

  Creates new bindings for the (already-existing) vars, with the
  supplied initial values, executes the exprs in an implicit do, then
  re-establishes the bindings that existed before.  The new bindings
  are made in parallel (unlike let); all init-exprs are evaluated
  before the vars are bound to their new values."
  {:added "1.0"}
  [bindings & body]
  (assert-args
    (vector? bindings) "a vector for its binding"
    (even? (count bindings)) "an even number of forms in binding vector")
  (let [var-ize (fn [var-vals]
                  (loop [ret [] vvs (seq var-vals)]
                    (if vvs
                      (recur  (conj (conj ret `(var ~(first vvs))) (second vvs))
                             (next (next vvs)))
                      (seq ret))))]
    `(with-bindings (hash-map ~@(var-ize bindings)) ~@body)))

(defn deref
  "Also reader macro: @var/@atom/@delay. When applied to a var or atom,
  returns its current state. When applied to a delay, forces
  it if not already forced."
  {:added "1.0"}
  [ref]
  (deref* ref))

(defn atom
  "Creates and returns an Atom with an initial value of x and zero or
  more options (in any order):

  :meta metadata-map

  :validator validate-fn

  If metadata-map is supplied, it will become the metadata on the
  atom. validate-fn must be nil or a side-effect-free fn of one
  argument, which will be passed the intended new state on any state
  change. If the new state is unacceptable, the validate-fn should
  return false or throw an exception."
  {:added "1.0"}
  [x & options]
  (apply atom* x options))

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

(defn remove
  "Returns a lazy sequence of the items in coll for which
  (pred item) returns false. pred must be free of side-effects."
  {:added "1.0"}
  [pred coll]
  (filter (complement pred) coll))

(defn take
  "Returns a lazy sequence of the first n items in coll, or all items if
  there are fewer than n."
  {:added "1.0"}
  [n coll]
  (lazy-seq
   (when (pos? n)
     (when-let [s (seq coll)]
       (cons (first s) (take (dec n) (rest s)))))))

(defn take-while
  "Returns a lazy sequence of successive items from coll while
  (pred item) returns true. pred must be free of side-effects."
  {:added "1.0"}
  [pred coll]
  (lazy-seq
   (when-let [s (seq coll)]
     (when (pred (first s))
       (cons (first s) (take-while pred (rest s)))))))

(defn drop
  "Returns a lazy sequence of all but the first n items in coll."
  {:added "1.0"}
  [n coll]
  (let [step (fn [n coll]
               (let [s (seq coll)]
                 (if (and (pos? n) s)
                   (recur (dec n) (rest s))
                   s)))]
    (lazy-seq (step n coll))))

(defn drop-last
  "Return a lazy sequence of all but the last n (default 1) items in coll"
  {:added "1.0"}
  ([s] (drop-last 1 s))
  ([n s] (map (fn [x _] x) s (drop n s))))

(defn take-last
  "Returns a seq of the last n items in coll.  Depending on the type
  of coll may be no better than linear time.  For vectors, see also subvec."
  {:added "1.0"}
  [n coll]
  (loop [s (seq coll), lead (seq (drop n coll))]
    (if lead
      (recur (next s) (next lead))
      s)))

(defn drop-while
  "Returns a lazy sequence of the items in coll starting from the first
  item for which (pred item) returns logical false."
  {:added "1.0"}
  [pred coll]
  (let [step (fn [pred coll]
               (let [s (seq coll)]
                 (if (and s (pred (first s)))
                   (recur pred (rest s))
                   s)))]
    (lazy-seq (step pred coll))))

(defn cycle
  "Returns a lazy (infinite!) sequence of repetitions of the items in coll."
  {:added "1.0"}
  [coll] (lazy-seq
          (when-let [s (seq coll)]
            (concat s (cycle s)))))

(defn split-at
  "Returns a vector of [(take n coll) (drop n coll)]"
  {:added "1.0"}
  [n coll]
  [(take n coll) (drop n coll)])

(defn split-with
  "Returns a vector of [(take-while pred coll) (drop-while pred coll)]"
  {:added "1.0"}
  [pred coll]
  [(take-while pred coll) (drop-while pred coll)])

(defn repeat
  "Returns a lazy (infinite!, or length n if supplied) sequence of xs."
  {:added "1.0"}
  ([x] (lazy-seq (cons x (repeat x))))
  ([n x] (take n (repeat x))))

(defn iterate
  "Returns a lazy sequence of x, (f x), (f (f x)) etc. f must be free of side-effects"
  {:added "1.0"}
  [f x] (cons x (lazy-seq (iterate f (f x)))))

(defn range
  "Returns a lazy seq of nums from start (inclusive) to end
  (exclusive), by step, where start defaults to 0, step to 1, and end to
  infinity. When step is equal to 0, returns an infinite sequence of
  start. When start is equal to end, returns empty list."
  {:added "1.0"}
  ([] (iterate inc 0))
  ([end] (range 0 end 1))
  ([start end] (range start end 1))
  ([start end step]
   (lazy-seq
    (let [comp (cond
                 (or (zero? step) (= start end)) not=
                 (pos? step) <
                 (neg? step) >)]
      (if (comp start end)
        (cons start (range (+ start step) end step))
        ())))))

(defn merge
  "Returns a map that consists of the rest of the maps conj-ed onto
  the first.  If a key occurs in more than one map, the mapping from
  the latter (left-to-right) will be the mapping in the result."
  {:added "1.0"}
  [& maps]
  (when (some identity maps)
    (reduce1 #(conj (or %1 {}) %2) maps)))

(defn merge-with
  "Returns a map that consists of the rest of the maps conj-ed onto
  the first.  If a key occurs in more than one map, the mapping(s)
  from the latter (left-to-right) will be combined with the mapping in
  the result by calling (f val-in-result val-in-latter)."
  {:added "1.0"}
  [f & maps]
  (when (some identity maps)
    (let [merge-entry (fn [m e]
                        (let [k (key e) v (val e)]
                          (if (contains? m k)
                            (assoc m k (f (get m k) v))
                            (assoc m k v))))
          merge2 (fn [m1 m2]
                   (reduce1 merge-entry (or m1 {}) (seq m2)))]
      (reduce1 merge2 maps))))

(defn zipmap
  "Returns a map with the keys mapped to the corresponding vals."
  {:added "1.0"}
  [keys vals]
  (loop [map {}
         ks (seq keys)
         vs (seq vals)]
    (if (and ks vs)
      (recur (assoc map (first ks) (first vs))
             (next ks)
             (next vs))
      map)))

(defmacro declare
  "defs the supplied var names with no bindings, useful for making forward declarations."
  {:added "1.0"}
  [& names] `(do ~@(map #(list 'def (vary-meta % assoc :declared true)) names)))

(defn sort
  "Returns a sorted sequence of the items in coll. If no comparator is
  supplied, uses compare."
  {:added "1.0"}
  ([coll]
   (sort compare coll))
  ([^Comparator comp coll]
   (sort* comp coll)))

(defn sort-by
  "Returns a sorted sequence of the items in coll, where the sort
  order is determined by comparing (keyfn item).  If no comparator is
  supplied, uses compare."
  {:added "1.0"}
  ([keyfn coll]
   (sort-by keyfn compare coll))
  ([keyfn ^Comparator comp coll]
   (sort (fn [x y] (comp (keyfn x) (keyfn y))) coll)))

(defn dorun
  "When lazy sequences are produced via functions that have side
  effects, any effects other than those needed to produce the first
  element in the seq do not occur until the seq is consumed. dorun can
  be used to force any effects. Walks through the successive nexts of
  the seq, does not retain the head and returns nil."
  {:added "1.0"}
  ([coll]
   (when (seq coll)
     (recur (next coll))))
  ([n coll]
   (when (and (seq coll) (pos? n))
     (recur (dec n) (next coll)))))

(defn doall
  "When lazy sequences are produced via functions that have side
  effects, any effects other than those needed to produce the first
  element in the seq do not occur until the seq is consumed. doall can
  be used to force any effects. Walks through the successive nexts of
  the seq, retains the head and returns it, thus causing the entire
  seq to reside in memory at one time."
  {:added "1.0"}
  ([coll]
   (dorun coll)
   coll)
  ([n coll]
   (dorun n coll)
   coll))

(defn nthnext
  "Returns the nth next of coll, (seq coll) when n is 0."
  {:added "1.0"}
  [coll n]
  (loop [n n xs (seq coll)]
    (if (and xs (pos? n))
      (recur (dec n) (next xs))
      xs)))

(defn nthrest
  "Returns the nth rest of coll, coll when n is 0."
  {:added "1.0"}
  [coll n]
  (loop [n n xs coll]
    (if (and (pos? n) (seq xs))
      (recur (dec n) (rest xs))
      xs)))

(defn partition
  "Returns a lazy sequence of lists of n items each, at offsets step
  apart. If step is not supplied, defaults to n, i.e. the partitions
  do not overlap. If a pad collection is supplied, use its elements as
  necessary to complete last partition upto n items. In case there are
  not enough padding elements, return a partition with less than n items."
  {:added "1.0"}
  ([n coll]
   (partition n n coll))
  ([n step coll]
   (lazy-seq
    (when-let [s (seq coll)]
      (let [p (doall (take n s))]
        (when (= n (count p))
          (cons p (partition n step (nthrest s step))))))))
  ([n step pad coll]
   (lazy-seq
    (when-let [s (seq coll)]
      (let [p (doall (take n s))]
        (if (= n (count p))
          (cons p (partition n step pad (nthrest s step)))
          (list (take n (concat p pad)))))))))

(defn eval
  "Evaluates the form data structure (not text!) and returns the result."
  {:added "1.0"}
  [form] (eval* form))

(defmacro doseq
  "Repeatedly executes body (presumably for side-effects) with
  bindings and filtering as provided by \"for\".  Does not retain
  the head of the sequence. Returns nil."
  {:added "1.0"}
  [seq-exprs & body]
  (assert-args
   (vector? seq-exprs) "a vector for its binding"
   (even? (count seq-exprs)) "an even number of forms in binding vector")
  (let [step (fn step [recform exprs]
               (if-not exprs
                 [true `(do ~@body)]
                 (let [k (first exprs)
                       v (second exprs)
                       seqsym (when-not (keyword? k) (gensym))
                       recform (if (keyword? k) recform `(recur (next ~seqsym)))
                       steppair (step recform (nnext exprs))
                       needrec (steppair 0)
                       subform (steppair 1)]
                   (cond
                     (= k :let) [needrec `(let ~v ~subform)]
                     (= k :while) [false `(when ~v
                                            ~subform
                                            ~@(when needrec [recform]))]
                     (= k :when) [false `(if ~v
                                           (do
                                             ~subform
                                             ~@(when needrec [recform]))
                                           ~recform)]
                     :else [true `(loop [~seqsym (seq ~v)]
                                    (when ~seqsym
                                      (let [~k (first ~seqsym)]
                                        ~subform
                                        ~@(when needrec [recform]))))]))))]
    (nth (step nil (seq seq-exprs)) 1)))

(defmacro dotimes
  "bindings => name n

  Repeatedly executes body (presumably for side-effects) with name
  bound to integers from 0 through n-1."
  {:added "1.0"}
  [bindings & body]
  (assert-args
   (vector? bindings) "a vector for its binding"
   (= 2 (count bindings)) "exactly 2 forms in binding vector")
  (let [i (first bindings)
        n (second bindings)]
    `(let [n# (int* ~n)]
       (loop [~i 0]
         (when (< ~i n#)
           ~@body
           (recur (inc ~i)))))))

(defn type
  "Returns the :type metadata of x, or its Type if none"
  {:added "1.0"}
  [x]
  (or (get (meta x) :type) (type* x)))

(defn num
  "Coerce to Number"
  {:tag Number
  :added "1.0"}
  [x] (num* x))

(defn double
  "Coerce to double"
  {:added "1.0"}
  [^Number x] (double* x))

(defn char
  "Coerce to char"
  {:added "1.0"}
  [x] (char* x))

(defn bool
  "Coerce to bool"
  {:added "1.0"}
  [x] (bool* x))

(defn number?
  "Returns true if x is a Number"
  {:added "1.0"}
  [x]
  (instance? Number x))

(defn mod
  "Modulus of num and div. Truncates toward negative infinity."
  {:added "1.0"}
  [num div]
  (let [m (rem num div)]
    (if (or (zero? m) (= (pos? num) (pos? div)))
      m
      (+ m div))))

(defn ratio?
  "Returns true if n is a Ratio"
  {:added "1.0"}
  [n] (instance? Ratio n))

(defn numerator
  "Returns the numerator part of a Ratio."
  {:tag BigInt
  :added "1.0"}
  [r]
  (numerator* r))

(defn denominator
  "Returns the denominator part of a Ratio."
  {:tag BigInt
  :added "1.0"}
  [r]
  (denominator* r))

(defn bigfloat?
  "Returns true if n is a BigFloat"
  {:added "1.0"}
  [n] (instance? BigFloat n))

(defn float?
  "Returns true if n is a floating point number"
  {:added "1.0"}
  [n]
  (instance? Double n))

(defn rational?
  "Returns true if n is a rational number"
  {:added "1.0"}
  [n]
  (or (integer? n) (ratio? n)))

(defn bigint
  "Coerce to BigInt"
  {:tag BigInt
  :added "1.0"}
  [x]
  (bigint* x))

(defn bigfloat
  "Coerce to BigFloat"
  {:tag BigFloat
  :added "1.0"}
  [x]
  (bigfloat* x))

(def
  ^{:arglists '([& args])
    :doc "Prints the object(s) to the output stream that is the current value
         of *out*.  Prints the object(s), separated by spaces if there is
         more than one.  By default, pr and prn print in a way that objects
         can be read by the reader"
         :added "1.0"}
  pr pr*)

(defn newline
  "Writes a platform-specific newline to *out*"
  {:added "1.0"}
  []
  (newline*))

(defn prn
  "Same as pr followed by (newline)."
  {:added "1.0"}
  [& more]
  (apply pr more)
  (newline))

(defn print
  "Prints the object(s) to the output stream that is the current value
  of *out*.  print and println produce output for human consumption."
  {:added "1.0"}
  [& more]
  (binding [*print-readably* nil]
    (apply pr more)))

(defn println
  "Same as print followed by (newline)"
  {:added "1.0"}
  [& more]
  (binding [*print-readably* nil]
    (apply prn more)))

(defn read
  "Reads the next object from reader (defaults to *in*)"
  {:added "1.0"}
  ([] (read *in*))
  ([reader] (read* reader)))

(def
  ^{:arglists '([])
    :doc "Reads the next line from *in*."
    :added "1.0"}
  read-line read-line*)

(def
  ^{:arglists '([s])
    :doc "Reads one object from the string s."
    :added "1.0"}
  read-string read-string*)

(defn subvec
  "Returns a persistent vector of the items in vector from
  start (inclusive) to end (exclusive).  If end is not supplied,
  defaults to (count vector). This operation is O(1) and very fast, as
  the resulting vector shares structure with the original and no
  trimming is done."
  {:added "1.0"}
  ([v start]
   (subvec v start (count v)))
  ([v start end]
   (subvec* v start end)))

(defmacro time
  "Evaluates expr and prints the time it took.  Returns the value of expr."
  {:added "1.0"}
  [expr]
  `(let [start# (nano-time*)
         ret# ~expr]
     (prn (str "Elapsed time: " (/ (double (- (nano-time*) start#)) 1000000.0) " msecs"))
     ret#))

(defn macroexpand-1
  "If form represents a macro form, returns its expansion, else returns form."
  {:added "1.0"}
  [form]
  (macroexpand-1* form))

(defn macroexpand
  "Repeatedly calls macroexpand-1 on form until it no longer
  represents a macro form, then returns it.  Note neither
  macroexpand-1 nor macroexpand expand macros in subforms."
  {:added "1.0"}
  [form]
  (let [ex (macroexpand-1 form)]
    (if (identical? ex form)
      form
      (macroexpand ex))))

(defn load-string
  "Sequentially read and evaluate the set of forms contained in the
  string"
  {:added "1.0"}
  [s]
  (load-string* s))

(defn set?
  "Returns true if x implements Set"
  {:added "1.0"}
  [x] (instance? Set x))

(defn set
  "Returns a set of the distinct elements of coll."
  {:added "1.0"}
  [coll]
  (if (set? coll)
    (with-meta coll nil)
    (reduce1 conj #{} coll)))

(defn ^:private filter-key
  [keyfn pred amap]
  (loop [ret {} es (seq amap)]
    (if es
      (if (pred (keyfn (first es)))
        (recur (assoc ret (key (first es)) (val (first es))) (next es))
        (recur ret (next es)))
      ret)))

(defn find-ns
  "Returns the namespace named by the symbol or nil if it doesn't exist."
  {:added "1.0"}
  [sym] (find-ns* sym))

(defn create-ns
  "Create a new namespace named by the symbol if one doesn't already
  exist, returns it or the already-existing namespace of the same
  name."
  {:added "1.0"}
  [sym] (create-ns* sym))

(defn remove-ns
  "Removes the namespace named by the symbol. Use with caution.
  Cannot be used to remove the clojure namespace."
  {:added "1.0"}
  [sym] (remove-ns* sym))

(defn all-ns
  "Returns a sequence of all namespaces."
  {:added "1.0"}
  [] (all-ns*))

(defn the-ns
  "If passed a namespace, returns it. Else, when passed a symbol,
  returns the namespace named by it, throwing an exception if not
  found."
  {:added "1.0"}
  ^Namespace [x]
  (if (instance? Namespace x)
    x
    (or (find-ns x) (throw (ex-info (str "No namespace: " x " found") {})))))

(defn ns-name
  "Returns the name of the namespace, a symbol."
  {:added "1.0"}
  [ns]
  (ns-name* (the-ns ns)))

(defn ns-map
  "Returns a map of all the mappings for the namespace."
  {:added "1.0"}
  [ns]
  (ns-map* (the-ns ns)))

(defn ns-unmap
  "Removes the mappings for the symbol from the namespace."
  {:added "1.0"}
  [ns sym]
  (ns-unmap* (the-ns ns) sym))

(defn ^:private public?
  [v]
  (not (:private (meta v))))

(defn ns-publics
  "Returns a map of the public intern mappings for the namespace."
  {:added "1.0"}
  [ns]
  (let [ns (the-ns ns)]
    (filter-key val (fn [^Var v] (and (instance? Var v)
                                      (= ns (var-ns* v))
                                      (public? v)))
                (ns-map ns))))

(defn ns-interns
  "Returns a map of the intern mappings for the namespace."
  {:added "1.0"}
  [ns]
  (let [ns (the-ns ns)]
    (filter-key val (fn [^Var v] (and (instance? Var v)
                                      (= ns (var-ns* v))))
                (ns-map ns))))

(defn refer
  "refers to all public vars of ns, subject to filters.
  filters can include at most one each of:

  :exclude list-of-symbols
  :only list-of-symbols
  :rename map-of-fromsymbol-tosymbol

  For each public interned var in the namespace named by the symbol,
  adds a mapping from the name of the var to the var to the current
  namespace.  Throws an exception if name is already mapped to
  something else in the current namespace. Filters can be used to
  select a subset, via inclusion or exclusion, or to provide a mapping
  to a symbol different from the var's name, in order to prevent
  clashes. Use :use in the ns macro in preference to calling this directly."
  {:added "1.0"}
  [ns-sym & filters]
  (let [ns (or (find-ns ns-sym) (throw (ex-info (str "No namespace: " ns-sym) {})))
        fs (apply hash-map filters)
        nspublics (ns-publics ns)
        rename (or (:rename fs) {})
        exclude (set (:exclude fs))
        to-do (if (= :all (:refer fs))
                (keys nspublics)
                (or (:refer fs) (:only fs) (keys nspublics)))]
    (when (and to-do (not (instance? Sequential to-do)))
      (throw (ex-info ":only/:refer value must be a sequential collection of symbols" {})))
    (doseq [sym to-do]
      (when-not (exclude sym)
        (let [v (nspublics sym)]
          (when-not v
            (throw (ex-info
                    (if (get (ns-interns ns) sym)
                      (str sym " is not public")
                      (str sym " does not exist")))))
          (refer* *ns* (or (rename sym) sym) v))))))

(defn ns-refers
  "Returns a map of the refer mappings for the namespace."
  {:added "1.0"}
  [ns]
  (let [ns (the-ns ns)]
    (filter-key val (fn [^Var v] (and (instance? Var v)
                                      (not= ns (var-ns* v))))
                (ns-map ns))))

(defn alias
  "Add an alias in the current namespace to another
  namespace. Arguments are two symbols: the alias to be used, and
  the symbolic name of the target namespace. Use :as in the ns macro in preference
  to calling this directly."
  {:added "1.0"}
  [alias namespace-sym]
  (alias* *ns* alias (the-ns namespace-sym)))

(defn ns-aliases
  "Returns a map of the aliases for the namespace."
  {:added "1.0"}
  [ns]
  (ns-aliases* (the-ns ns)))

(defn ns-unalias
  "Removes the alias for the symbol from the namespace."
  {:added "1.0"}
  [ns sym]
  (ns-unalias* (the-ns ns) sym))

(defn take-nth
  "Returns a lazy seq of every nth item in coll."
  {:added "1.0"}
  [n coll]
  (lazy-seq
   (when-let [s (seq coll)]
     (cons (first s) (take-nth n (drop n s))))))

(defn interleave
  "Returns a lazy seq of the first item in each coll, then the second etc."
  {:added "1.0"}
  ([] ())
  ([c1] (lazy-seq c1))
  ([c1 c2]
   (lazy-seq
    (let [s1 (seq c1) s2 (seq c2)]
      (when (and s1 s2)
        (cons (first s1) (cons (first s2)
                               (interleave (rest s1) (rest s2))))))))
  ([c1 c2 & colls]
   (lazy-seq
    (let [ss (map seq (conj colls c2 c1))]
      (when (every? identity ss)
        (concat (map first ss) (apply interleave (map rest ss))))))))

(defn ns-resolve
  "Returns the var or type to which a symbol will be resolved in the
  namespace (unless found in the environment), else nil.  Note that
  if the symbol is fully qualified, the var/Class to which it resolves
  need not be present in the namespace."
  {:added "1.0"}
  ([ns sym]
    (ns-resolve ns nil sym))
  ([ns env sym]
    (when-not (contains? env sym)
      (ns-resolve* (the-ns ns) sym))))

(defn resolve
  "Same as (ns-resolve *ns* sym) or (ns-resolve *ns* env sym)"
  {:added "1.0"}
  ([sym] (ns-resolve *ns* sym))
  ([env sym] (ns-resolve *ns* env sym)))

(def
  ^{:arglists '([& keyvals])
    :doc "Constructs an array-map. If any keys are equal, they are handled as
         if by repeated uses of assoc."
         :added "1.0"}
  array-map array-map*)

;redefine let and loop  with destructuring
(defn destructure [bindings]
  (let [bents (partition 2 bindings)
        pb (fn pb [bvec b v]
             (let [pvec
                   (fn [bvec b val]
                     (let [gvec (gensym "vec__")]
                       (loop [ret (-> bvec (conj gvec) (conj val))
                              n 0
                              bs b
                              seen-rest? false]
                         (if (seq bs)
                           (let [firstb (first bs)]
                             (cond
                               (= firstb '&) (recur (pb ret (second bs) (list `nthnext gvec n))
                                                    n
                                                    (nnext bs)
                                                    true)
                               (= firstb :as) (pb ret (second bs) gvec)
                               :else (if seen-rest?
                                       (throw (ex-info "Unsupported binding form, only :as can follow & parameter" {}))
                                       (recur (pb ret firstb  (list `nth gvec n nil))
                                              (inc n)
                                              (next bs)
                                              seen-rest?))))
                           ret))))
                   pmap
                   (fn [bvec b v]
                     (let [gmap (gensym "map__")
                           gmapseq (with-meta gmap {:tag 'Seq})
                           defaults (:or b)]
                       (loop [ret (-> bvec (conj gmap) (conj v)
                                      (conj gmap) (conj `(if (seq? ~gmap) (apply array-map* (seq ~gmapseq)) ~gmap))
                                      ((fn [ret]
                                         (if (:as b)
                                           (conj ret (:as b) gmap)
                                           ret))))
                              bes (reduce1
                                   (fn [bes entry]
                                     (reduce1 #(assoc %1 %2 ((val entry) %2))
                                              (dissoc bes (key entry))
                                              ((key entry) bes)))
                                   (dissoc b :as :or)
                                   {:keys #(if (keyword? %) % (keyword (str %))),
                                    :strs str, :syms #(list `quote %)})]
                         (if (seq bes)
                           (let [bb (key (first bes))
                                 bk (val (first bes))
                                 bv (if (contains? defaults bb)
                                      (list `get gmap bk (defaults bb))
                                      (list `get gmap bk))]
                             (recur (cond
                                      (symbol? bb) (-> ret (conj (if (namespace bb) (symbol (name bb)) bb)) (conj bv))
                                      (keyword? bb) (-> ret (conj (symbol (name bb)) bv))
                                      :else (pb ret bb bv))
                                    (next bes)))
                           ret))))]
               (cond
                 (symbol? b) (-> bvec (conj b) (conj v))
                 (vector? b) (pvec bvec b v)
                 (map? b) (pmap bvec b v)
                 :else (throw (ex-info (str "Unsupported binding form: " b) {})))))
        process-entry (fn [bvec b] (pb bvec (first b) (second b)))]
    (if (every? symbol? (map first bents))
      bindings
      (reduce1 process-entry [] bents))))

(defmacro let
  "binding => binding-form init-expr

  Evaluates the exprs in a lexical context in which the symbols in
  the binding-forms are bound to their respective init-exprs or parts
  therein."
  {:added "1.0", :special-form true, :forms '[(let [bindings*] exprs*)]}
  [bindings & body]
  (assert-args
   (vector? bindings) "a vector for its binding"
   (even? (count bindings)) "an even number of forms in binding vector")
  `(let* ~(destructure bindings) ~@body))

(defn ^{:private true}
  maybe-destructured
  [params body]
  (if (every? symbol? params)
    (cons params body)
    (loop [params params
           new-params (with-meta [] (meta params))
           lets []]
      (if params
        (if (symbol? (first params))
          (recur (next params) (conj new-params (first params)) lets)
          (let [gparam (gensym "p__")]
            (recur (next params) (conj new-params gparam)
                   (-> lets (conj (first params)) (conj gparam)))))
        `(~new-params
          (let ~lets
            ~@body))))))

;redefine fn with destructuring and pre/post conditions
(defmacro fn
  "params => positional-params* , or positional-params* & next-param
  positional-param => binding-form
  next-param => binding-form
  name => symbol

  Defines a function"
  {:added "1.0", :special-form true,
   :forms '[(fn name? [params* ] exprs*) (fn name? ([params* ] exprs*)+)]}
  [& sigs]
    (let [name (if (symbol? (first sigs)) (first sigs) nil)
          sigs (if name (next sigs) sigs)
          sigs (if (vector? (first sigs))
                 (list sigs)
                 (if (seq? (first sigs))
                   sigs
                   ;; Assume single arity syntax
                   (throw (ex-info
                            (if (seq sigs)
                              (str "Parameter declaration "
                                   (first sigs)
                                   " should be a vector")
                              (str "Parameter declaration missing"))
                            {}))))
          psig (fn* [sig]
                 ;; Ensure correct type before destructuring sig
                 (when (not (seq? sig))
                   (throw (ex-info
                            (str "Invalid signature " sig
                                 " should be a list")
                            {})))
                 (let [[params & body] sig
                       _ (when (not (vector? params))
                           (throw (ex-info
                                    (if (seq? (first sigs))
                                      (str "Parameter declaration " params
                                           " should be a vector")
                                      (str "Invalid signature " sig
                                           " should be a list"))
                                    {})))
                       conds (when (and (next body) (map? (first body)))
                                           (first body))
                       body (if conds (next body) body)
                       conds (or conds (meta params))
                       pre (:pre conds)
                       post (:post conds)
                       body (if post
                              `((let [~'% ~(if (< 1 (count body))
                                            `(do ~@body)
                                            (first body))]
                                 ~@(map (fn* [c] `(assert ~c)) post)
                                 ~'%))
                              body)
                       body (if pre
                              (concat (map (fn* [c] `(assert ~c)) pre)
                                      body)
                              body)]
                   (maybe-destructured params body)))
          new-sigs (map psig sigs)]
      (with-meta
        (if name
          (list* 'fn* name new-sigs)
          (cons 'fn* new-sigs))
        (meta &form))))

(defmacro loop
  "Evaluates the exprs in a lexical context in which the symbols in
  the binding-forms are bound to their respective init-exprs or parts
  therein. Acts as a recur target."
  {:added "1.0", :special-form true, :forms '[(loop [bindings*] exprs*)]}
  [bindings & body]
  (assert-args
   (vector? bindings) "a vector for its binding"
   (even? (count bindings)) "an even number of forms in binding vector")
  (let [db (destructure bindings)]
    (if (= db bindings)
      `(loop* ~bindings ~@body)
      (let [vs (take-nth 2 (drop 1 bindings))
            bs (take-nth 2 bindings)
            gs (map (fn [b] (if (symbol? b) b (gensym))) bs)
            bfs (reduce1 (fn [ret [b v g]]
                           (if (symbol? b)
                             (conj ret g v)
                             (conj ret g v b g)))
                         [] (map vector bs vs gs))]
        `(let ~bfs
           (loop* ~(vec (interleave gs gs))
                  (let ~(vec (interleave bs gs))
                    ~@body)))))))

(defmacro when-first
  "bindings => x xs

  Roughly the same as (when (seq xs) (let [x (first xs)] body)) but xs is evaluated only once"
  {:added "1.0"}
  [bindings & body]
  (assert-args
   (vector? bindings) "a vector for its binding"
   (= 2 (count bindings)) "exactly 2 forms in binding vector")
  (let [[x xs] bindings]
    `(when-let [xs# (seq ~xs)]
       (let [~x (first xs#)]
         ~@body))))

(defmacro lazy-cat
  "Expands to code which yields a lazy sequence of the concatenation
  of the supplied colls.  Each coll expr is not evaluated until it is
  needed.

  (lazy-cat xs ys zs) === (concat (lazy-seq xs) (lazy-seq ys) (lazy-seq zs))"
  {:added "1.0"}
  [& colls]
  `(concat ~@(map #(list `lazy-seq %) colls)))

(defmacro for
  "List comprehension. Takes a vector of one or more
  binding-form/collection-expr pairs, each followed by zero or more
  modifiers, and yields a lazy sequence of evaluations of expr.
  Collections are iterated in a nested fashion, rightmost fastest,
  and nested coll-exprs can refer to bindings created in prior
  binding-forms.  Supported modifiers are: :let [binding-form expr ...],
  :while test, :when test.

  (take 100 (for [x (range 100000000) y (range 1000000) :while (< y x)]  [x y]))"
  [seq-exprs body-expr]
  (assert-args for
               (vector? seq-exprs) "a vector for its binding"
               (even? (count seq-exprs)) "an even number of forms in binding vector")
  (let [to-groups (fn [seq-exprs]
                    (reduce1 (fn [groups [k v]]
                               (if (keyword? k)
                                 (conj (pop groups) (conj (peek groups) [k v]))
                                 (conj groups [k v])))
                             [] (partition 2 seq-exprs)))
        err (fn [& msg] (throw (ex-info (apply str msg) {})))
        emit-bind (fn emit-bind [[[bind expr & mod-pairs]
                                  & [[_ next-expr] :as next-groups]]]
                    (let [giter (gensym "iter__")
                          gxs (gensym "s__")
                          do-mod (fn do-mod [[[k v :as pair] & etc]]
                                   (cond
                                     (= k :let) `(let ~v ~(do-mod etc))
                                     (= k :while) `(when ~v ~(do-mod etc))
                                     (= k :when) `(if ~v
                                                    ~(do-mod etc)
                                                    (recur (rest ~gxs)))
                                     (keyword? k) (err "Invalid 'for' keyword " k)
                                     next-groups
                                     `(let [iterys# ~(emit-bind next-groups)
                                            fs# (seq (iterys# ~next-expr))]
                                        (if fs#
                                          (concat fs# (~giter (rest ~gxs)))
                                          (recur (rest ~gxs))))
                                     :else `(cons ~body-expr
                                                  (~giter (rest ~gxs)))))]
                      `(fn ~giter [~gxs]
                         (lazy-seq
                          (loop [~gxs ~gxs]
                            (when-first [~bind ~gxs]
                              ~(do-mod mod-pairs)))))))]
    `(let [iter# ~(emit-bind (to-groups seq-exprs))]
       (iter# ~(second seq-exprs)))))

(defmacro comment
  "Ignores body, yields nil"
  {:added "1.0"}
  [& body])

(defmacro with-out-str
  "Evaluates exprs in a context in which *out* is bound to a fresh
  Buffer.  Returns the string created by any nested printing
  calls."
  {:added "1.0"}
  [& body]
  `(binding [*out* (buffer*)]
     ~@body
     (str *out*)))

(defmacro with-in-str
  "Evaluates body in a context in which *in* is bound to a fresh
  Buffer initialized with the string s."
  {:added "1.0"}
  [s & body]
  `(binding [*in* (buffer* ~s)]
     ~@body))

(defn pr-str
  "pr to a string, returning it"
  {:tag String
  :added "1.0"}
  [& xs]
  (with-out-str
    (apply pr xs)))

(defn prn-str
  "prn to a string, returning it"
  {:tag String
  :added "1.0"}
  [& xs]
  (with-out-str
    (apply prn xs)))

(defn print-str
  "print to a string, returning it"
  {:tag String
  :added "1.0"}
  [& xs]
  (with-out-str
    (apply print xs)))

(defn println-str
  "println to a string, returning it"
  {:tag String
  :added "1.0"}
  [& xs]
  (with-out-str
    (apply println xs)))

(defn ex-data
  "Returns exception data (a map) if ex is an ExInfo.
  Otherwise returns nil."
  {:added "1.0"}
  [ex]
  (when (instance? ExInfo ex)
    (ex-data* ^ExInfo ex)))

(defn hash
  "Returns the hash code of its argument."
  {:added "1.0"}
  [x] (hash* x))

(defmacro assert
  "Evaluates expr and throws an exception if it does not evaluate to
  logical true."
  {:added "1.0"}
  ([x]
   (when *assert*
     `(when-not ~x
        (throw (ex-info (str "Assert failed: " '~x) {})))))
  ([x message]
   (when *assert*
     `(when-not ~x
        (throw (ex-info (str "Assert failed: " ~message "\n" '~x)))))))

(defn test
  "test [v] finds fn at key :test in var metadata and calls it,
  presuming failure will throw exception"
  {:added "1.0"}
  [v]
  (let [f (:test (meta v))]
    (if f
      (do (f) :ok)
      :no-test)))

(defn re-pattern
  "Returns an instance of Regex"
  {:tag Regex
  :added "1.0"}
  [s]
  (if (instance? Regex s)
    s
    (regex* s)))

(defn re-seq
  "Returns a sequence of successive matches of pattern in string"
  {:added "1.0"}
  [^Regex re s]
  (re-seq* re s))

(defn re-find
  "Returns the leftmost regex match, if any, of string to pattern."
  {:added "1.0"}
  [^Regex re s]
  (re-find* re s))

(defn re-matches
  "Returns the match, if any, of string to pattern."
  {:added "1.0"}
  [^Regex re s]
  (let [m (re-find re s)
        c (if (instance? String m)
            (count m)
            (count (first m)))]
    (when (= c (count s))
      m)))

(defn rand
  "Returns a random floating point number between 0 (inclusive) and
  n (default 1) (exclusive)."
  {:added "1.0"}
  ([] (rand*))
  ([n] (* n (rand))))

(defn rand-int
  "Returns a random integer between 0 (inclusive) and n (exclusive)."
  {:added "1.0"}
  [n] (int (rand n)))

(defmacro defn-
  "same as defn, yielding non-public def"
  {:added "1.0"}
  [name & decls]
  (list* `defn (with-meta name (assoc (meta name) :private true)) decls))

(defn tree-seq
  "Returns a lazy sequence of the nodes in a tree, via a depth-first walk.
  branch? must be a fn of one arg that returns true if passed a node
  that can have children (but may not).  children must be a fn of one
  arg that returns a sequence of the children. Will only be called on
  nodes for which branch? returns true. Root is the root node of the
  tree."
  {:added "1.0"}
  [branch? children root]
  (let [walk (fn walk [node]
               (lazy-seq
                (cons node
                      (when (branch? node)
                        (mapcat walk (children node))))))]
    (walk root)))

; TODO:
; (defn file-seq
;   "A tree seq on directory"
;   {:added "1.0"}
;   [dir]
;   (tree-seq
;    (fn [^java.io.File f] (. f (isDirectory)))
;    (fn [^java.io.File d] (seq (. d (listFiles))))
;    dir))

(defn xml-seq
  "A tree seq on the xml elements as per xml/parse"
  {:added "1.0"}
  [root]
  (tree-seq
   (complement string?)
   (comp seq :content)
   root))

(defn special-symbol?
  "Returns true if s names a special form"
  {:added "1.0"}
  [s]
  (special-symbol?* s))

(defn var?
  "Returns true if v is of type Var"
  {:added "1.0"}
  [v] (instance? Var v))

(defn subs
  "Returns the substring of s beginning at start inclusive, and ending
  at end (defaults to length of string), exclusive."
  {:added "1.0"}
  (^String [^String s start] (subs* s start))
  (^String [^String s start end] (subs* s start end)))

(defn max-key
  "Returns the x for which (k x), a number, is greatest."
  {:added "1.0"}
  ([k x] x)
  ([k x y] (if (> (k x) (k y)) x y))
  ([k x y & more]
   (reduce1 #(max-key k %1 %2) (max-key k x y) more)))

(defn min-key
  "Returns the x for which (k x), a number, is least."
  {:added "1.0"}
  ([k x] x)
  ([k x y] (if (< (k x) (k y)) x y))
  ([k x y & more]
   (reduce1 #(min-key k %1 %2) (min-key k x y) more)))

(defn distinct
  "Returns a lazy sequence of the elements of coll with duplicates removed."
  {:added "1.0"}
  [coll]
  (let [step (fn step [xs seen]
               (lazy-seq
                ((fn [[f :as xs] seen]
                   (when-let [s (seq xs)]
                     (if (contains? seen f)
                       (recur (rest s) seen)
                       (cons f (step (rest s) (conj seen f))))))
                 xs seen)))]
    (step coll #{})))

(defn replace
  "Given a map of replacement pairs and a vector/collection, returns a
  vector/seq with any elements = a key in smap replaced with the
  corresponding val in smap."
  {:added "1.0"}
  [smap coll]
  (if (vector? coll)
    (reduce1 (fn [v i]
               (if-let [e (find smap (nth v i))]
                 (assoc v i (val e))
                 v))
             coll (range (count coll)))
    (map #(if-let [e (find smap %)] (val e) %) coll)))

(defn get-in
  "Returns the value in a nested associative structure,
  where ks is a sequence of keys. Returns nil if the key
  is not present, or the not-found value if supplied."
  {:added "1.0"}
  ([m ks]
   (reduce1 get m ks))
  ([m ks not-found]
   (loop [sentinel {}
          m m
          ks (seq ks)]
     (if ks
       (let [m (get m (first ks) sentinel)]
         (if (identical? sentinel m)
           not-found
           (recur sentinel m (next ks))))
       m))))

(defn assoc-in
  "Associates a value in a nested associative structure, where ks is a
  sequence of keys and v is the new value and returns a new nested structure.
  If any levels do not exist, hash-maps will be created."
  {:added "1.0"}
  [m [k & ks] v]
  (if ks
    (assoc m k (assoc-in (get m k) ks v))
    (assoc m k v)))

(defn update-in
  "'Updates' a value in a nested associative structure, where ks is a
  sequence of keys and f is a function that will take the old value
  and any supplied args and return the new value, and returns a new
  nested structure.  If any levels do not exist, hash-maps will be
  created."
  {:added "1.0"}
  ([m [k & ks] f & args]
   (if ks
     (assoc m k (apply update-in (get m k) ks f args))
     (assoc m k (apply f (get m k) args)))))

(defn update
  "'Updates' a value in an associative structure, where k is a
  key and f is a function that will take the old value
  and any supplied args and return the new value, and returns a new
  structure.  If the key does not exist, nil is passed as the old value."
  {:added "1.0"}
  ([m k f]
   (assoc m k (f (get m k))))
  ([m k f x]
   (assoc m k (f (get m k) x)))
  ([m k f x y]
   (assoc m k (f (get m k) x y)))
  ([m k f x y z]
   (assoc m k (f (get m k) x y z)))
  ([m k f x y z & more]
   (assoc m k (apply f (get m k) x y z more))))

(defn coll?
  "Returns true if x implements Collection"
  {:added "1.0"}
  [x] (instance? Collection x))

(defn list?
  "Returns true if x is a List"
  {:added "1.0"}
  [x] (instance? List x))

(defn callable?
  "Returns true if x implements Callable. Note that many data structures
  (e.g. sets and maps) implement Callable."
  {:added "1.0"}
  [x] (instance? Callable x))

(defn fn?
  "Returns true if x is Fn, i.e. is an object created via fn."
  {:added "1.0"}
  [x] (instance? Fn x))

(defn associative?
  "Returns true if coll implements Associative"
  {:added "1.0"}
  [coll] (instance? Associative coll))

(defn sequential?
  "Returns true if coll implements Sequential"
  {:added "1.0"}
  [coll] (instance? Sequential coll))

(defn counted?
  "Returns true if coll implements count in constant time"
  {:added "1.0"}
  [coll] (instance? Counted coll))

(defn reversible?
  "Returns true if coll implements Reversible"
  {:added "1.0"}
  [coll] (instance? Reversible coll))

(def
  ^{:doc "bound in a repl to the most recent value printed"
    :added "1.0"}
  *1)

(def
  ^{:doc "bound in a repl to the second most recent value printed"
    :added "1.0"}
  *2)

(def
  ^{:doc "bound in a repl to the third most recent value printed"
    :added "1.0"}
  *3)

(def
  ^{:doc "bound in a repl to the most recent exception caught by the repl"
    :added "1.0"}
  *e)

(defn trampoline
  "trampoline can be used to convert algorithms requiring mutual
  recursion without stack consumption. Calls f with supplied args, if
  any. If f returns a fn, calls that fn with no arguments, and
  continues to repeat, until the return value is not a fn, then
  returns that non-fn value. Note that if you want to return a fn as a
  final value, you must wrap it in some data structure and unpack it
  after trampoline returns."
  {:added "1.0"}
  ([f]
   (let [ret (f)]
     (if (fn? ret)
       (recur ret)
       ret)))
  ([f & args]
   (trampoline #(apply f args))))

(defn intern
  "Finds or creates a var named by the symbol name in the namespace
  ns (which can be a symbol or a namespace), setting its root binding
  to val if supplied. The namespace must exist. The var will adopt any
  metadata from the name symbol.  Returns the var."
  {:added "1.0"}
  ([ns ^Symbol name]
   (let [v (intern* (the-ns ns) name)]
     (when (meta name) (set-meta* v (meta name)))
     v))
  ([ns name val]
   (let [v (intern* (the-ns ns) name val)]
     (when (meta name) (set-meta* v (meta name)))
     v)))

(defmacro while
  "Repeatedly executes body while test expression is true. Presumes
  some side-effect will cause test to become false/nil. Returns nil"
  {:added "1.0"}
  [test & body]
  `(loop []
     (when ~test
       ~@body
       (recur))))

; (defn memoize
;   "Returns a memoized version of a referentially transparent function. The
;   memoized version of the function keeps a cache of the mapping from arguments
;   to results and, when calls with the same arguments are repeated often, has
;   higher performance at the expense of higher memory use."
;   {:added "1.0"}
;   [f]
;   (let [mem (atom {})]
;     (fn [& args]
;       (if-let [e (find @mem args)]
;         (val e)
;         (let [ret (apply f args)]
;           (swap! mem assoc args ret)
;           ret)))))

(defn empty?
  "Returns true if coll has no items - same as (not (seq coll)).
  Please use the idiom (seq x) rather than (not (empty? x))"
  {:added "1.0"}
  [coll] (not (seq coll)))

(defmacro cond->
  "Takes an expression and a set of test/form pairs. Threads expr (via ->)
  through each form for which the corresponding test
  expression is true. Note that, unlike cond branching, cond-> threading does
  not short circuit after the first true test expression."
  {:added "1.0"}
  [expr & clauses]
  (assert (even? (count clauses)))
  (let [g (gensym)
        steps (map (fn [[test step]] `(if ~test (-> ~g ~step) ~g))
                   (partition 2 clauses))]
    `(let [~g ~expr
           ~@(interleave (repeat g) (butlast steps))]
       ~(if (empty? steps)
          g
          (last steps)))))

(defmacro cond->>
  "Takes an expression and a set of test/form pairs. Threads expr (via ->>)
  through each form for which the corresponding test expression
  is true.  Note that, unlike cond branching, cond->> threading does not short circuit
  after the first true test expression."
  {:added "1.0"}
  [expr & clauses]
  (assert (even? (count clauses)))
  (let [g (gensym)
        steps (map (fn [[test step]] `(if ~test (->> ~g ~step) ~g))
                   (partition 2 clauses))]
    `(let [~g ~expr
           ~@(interleave (repeat g) (butlast steps))]
       ~(if (empty? steps)
          g
          (last steps)))))

(defmacro as->
  "Binds name to expr, evaluates the first form in the lexical context
  of that binding, then binds name to that result, repeating for each
  successive form, returning the result of the last form."
  {:added "1.0"}
  [expr name & forms]
  `(let [~name ~expr
         ~@(interleave (repeat name) (butlast forms))]
     ~(if (empty? forms)
        name
        (last forms))))

(defmacro some->
  "When expr is not nil, threads it into the first form (via ->),
  and when that result is not nil, through the next etc"
  {:added "1.0"}
  [expr & forms]
  (let [g (gensym)
        steps (map (fn [step] `(if (nil? ~g) nil (-> ~g ~step)))
                   forms)]
    `(let [~g ~expr
           ~@(interleave (repeat g) (butlast steps))]
       ~(if (empty? steps)
          g
          (last steps)))))

(defmacro some->>
  "When expr is not nil, threads it into the first form (via ->>),
  and when that result is not nil, through the next etc"
  {:added "1.0"}
  [expr & forms]
  (let [g (gensym)
        steps (map (fn [step] `(if (nil? ~g) nil (->> ~g ~step)))
                   forms)]
    `(let [~g ~expr
           ~@(interleave (repeat g) (butlast steps))]
       ~(if (empty? steps)
          g
          (last steps)))))

(defn keep
  "Returns a lazy sequence of the non-nil results of (f item). Note,
  this means false return values will be included.  f must be free of
  side-effects."
  {:added "1.0"}
  [f coll]
  (lazy-seq
   (when-let [s (seq coll)]
     (let [x (f (first s))]
       (if (nil? x)
         (keep f (rest s))
         (cons x (keep f (rest s))))))))

(defn slurp
  "Opens a file f and reads all its contents, returning a string."
  {:added "1.0"}
  [f]
  (slurp* f))
