;; Additional restriction on arities where it makes sense

(def __=__ =)
(defn =
  ^Boolean [x y & more]
  (apply __=__ x y more))

(def __not=__ not=)
(defn not=
  ^Boolean [x y & more]
  (apply __not=__ x y more))

(def __<__ <)
(defn <
  ^Boolean [^Number x ^Number y & more]
  (apply __<__ x y more))

(def __>__ >)
(defn >
  ^Boolean [^Number x ^Number y & more]
  (apply __>__ x y more))

(def __<=__ <=)
(defn <=
  ^Boolean [^Number x ^Number y & more]
  (apply __<=__ x y more))

(def __>=__ >=)
(defn >=
  ^Boolean [^Number x ^Number y & more]
  (apply __>=__ x y more))

(def __==__ ==)
(defn ==
  ^Boolean [^Number x ^Number y & more]
  (apply __==__ x y more))

(def __merge__ merge)
(defn merge
  ^Map [^Map m1 & more]
  (apply __merge__ m1 more))

(def __merge-with__ merge-with)
(defn merge-with
  ^Map [^Callable f ^Map m1 & more]
  (apply __merge-with__ f m1 more))

;; Redefine when and when-not with additional checks

(defmacro when
  "Evaluates test. If logical true, evaluates body in an implicit do."
  {:added "1.0"}
  [test & body]
  (let [c (count body)
        b (if (> c 1)
            (cons 'do body)
            (first body))]
    (when *linter-mode*
      (when (zero? c)
        (println-linter__ (ex-info "when form with empty body" {:form &form :_prefix "Parse warning"}))))
    (list 'if test b nil)))

(defmacro when-not
  "Evaluates test. If logical false, evaluates body in an implicit do."
  {:added "1.0"}
  [test & body]
  (let [c (count body)
        b (if (> c 1)
            (cons 'do body)
            (first body))]
    (when *linter-mode*
      (when (zero? c)
        (println-linter__ (ex-info "when-not form with empty body" {:form &form :_prefix "Parse warning"}))))
    (list 'if test nil b)))
