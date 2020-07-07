(if 1 2 3)

(if 1
  2
  3)

(if 1 2)

(if 1
  2)

(if 1)

(if)

(fn [a b] 1 2)

(fn [a b]
  1
  2)

(fn a [a b] 1 2)

(fn a [a b]
  1
  2)

(let [a 1 b 2] 1 2)

(let [a 1
      b 2]
  1
  2)

(let [] 1)

(let []
  1)

(let [a 1] 1 2)

(let [a 1]
  1
  2)

(let [a]
  1
  2)

(let [a] 1 2)

(loop [a 1 b 1] (if (> a 10) [a b] (recur (inc a) (dec b))))

(loop [a 1
       b 1]
  (if (> a 10)
    [a b]
    (recur (inc a) (dec b))))

(letfn [(neven? [n] (if (zero? n) true (nodd? (dec n))))
        (nodd? [n] (if (zero? n) false (neven? (dec n))))]
  (neven? 10))

(letfn [(neven? [n] (if (zero? n)
                      true
                      (nodd? (dec n))))
        (nodd? [n] (if (zero? n)
                     false
                     (neven? (dec n))))]
  (neven? 10))

(do 1 2)

(do
  1
  2)

(do 1
    2)

(try 1 (catch Exception e 2) (finally 3))

(try
  1
  (catch Exception e
    2)
  (finally
    3))

(defn plus [x y] (+ x y))

(defn plus
  [x y]
  (+ x y))

(defn cast
  "Throws an error if x is not of a type t, else returns x."
  {:added "1.0"}
  [t x]
  (cast__ t x))

(deftest t (is (= 1 2)))

(deftest t
  (is (= 1 2)))

(def PI 3.14)

(def PI
  3.14)

(ns my.test
  (:require my.test1
            [my.test2]
            [my.test3 :as test3 :refer [f1]])
  (:import (java.time LocalDateTime ZonedDateTime ZoneId)
           java.time.format.DateTimeFormatter))

(defn test-docstring
  "Given a multimethod and a dispatch value, returns the dispatch fn
  that would apply to that value, or nil if none apply and no default"
  {:added "1.0"}
  [t]
  (print "ha
         ha"))

(test-call 1 2 3)

(test-call 1
           2
           3)

(test-call 1 2
           3)

(test-call
 1
 2
 3)

(test-call
 1 2
 3)

@mfatom

@1

#"t"

#'t

#^:t []

#(inc %)

'test

'[test]

`(if-not ~test ~then nil)

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

(def ^{:arglists '([& items])
       :doc "Creates a new list containing the items."
       :added "1.0"
       :tag List}
  list list__)

{1 2 3 4}

{1 2
 3 4}

[1 2
 3 4]

#{1 2
  3 4 5}

[#?(:cljs 1)]

(#?(:cljs 1))

#?(:clj 1)

#?@(:cljs 3)

(def regexp #?(:clj re-pattern :cljs js/XRegExp))

#?(:cljs)

#?(:cljs (let [] 1) :default (let [] 1))

[#?@(:clj 1)]

#:t{:g 1}

#::{:g 1}

#:t{:_/g 1}

#:t{:h/g 1}

#::s{:g 1}

#::{g 1}

#inst 1

#uuid 2

#t 4

#g [a]

(defn ^:private line-seq*
  [rdr]
  (when-let [line (reader-read-line__ rdr)]
    (cons line (lazy-seq (line-seq* rdr)))))

(defrecord StandardInterceptor [name request response]
  Interceptor
  (-process-request [{:keys [request]} opts]
    (request opts))
  (-process-response [{:keys [response]} xhrio]
    (response xhrio)))

(defprotocol AjaxRequest
  "An abstraction for a running ajax request."
  (-abort [this]
    "Aborts a running ajax request, if possible."))

(extend-protocol IComparable
  Symbol
  (-compare [x y]
    (if (symbol? y)
      (compare-symbols x y)
      (throw (js/Error. (str "Cannot compare " x " to " y)))))

  Keyword
  (-compare [x y]
    (if (keyword? y)
      (compare-keywords x y)
      (throw (js/Error. (str "Cannot compare " x " to " y))))))

(defmulti ^:private render-at-rule
  "Render a CSS at-rule"
  :identifier)

(defmethod render-at-rule :default [_] nil)

(defmethod render-at-rule :import
  [{:keys [value]}]
  (let [{:keys [url media-queries]} value
        url (if (string? url)
              (util/wrap-quotes url)
              (render-css url))
        queries (when media-queries
                  (render-media-expr media-queries))]
    (str "@import "
         (if queries (str url " " queries) url)
         semicolon)))

; test comment

;; another one

; multiline comment
; test

(def s
  "foo
  bar")

;; TODO: something
(defn ^:private foo
  "Some useful
  docstring." ; random comment
  ;; Another random comment
  []
  ; Comment inside function body
  (+ 1 2) ; end of line comment

  ;; Return nil
  nil)

#_(+ 1 2)

(comment (+ 1 2))

(= "sadf" {:foo "sadf"
           :bar "qewr"})

{:foo "sdfsdf"
 ;;some comment
 :bar "1234"}

{:foo "sdfsdf" ;;some comment
 ;; test
 :bar 1234
 ;;some comment
 ; :bar "1234"
 }

(cond
  (= 1 2) 1
  (= 2 2) 3

  :else
  (println "ha"))

(cond1-> t
         (= 1 2) (assoc :t 1)
         true identity)

(defn- print-char [c]
  (-write *out* (condp = c
                  \backspace "\\backspace"
                  \tab "\\tab"
                  \newline "\\newline"
                  \formfeed "\\formfeed"
                  \return "\\return"
                  \" "\\\""
                  \\ "\\\\"
                  (str "\\" c))))

(use-fixtures :once
  my-fixture)

(with-something
  do-something)

(foor (test-call
       a
       ;;asdfds
       )
      1)

[1

 ;;sdf
 2
 ;;test
 ]

#{1
  ;sdf
  2
  ;;test
  }

{:t 1
 #_1
 }

{:t ^:g []
 :g ^:g []}

{:t
 ;;test
 1
 #_1 2 3}

#"(?<=[a-z])(?=[A-Z])"
