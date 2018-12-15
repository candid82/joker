;this example illustrates that the dispatch type
;does not have to be a symbol, but can be anything (in this case, it's a string)

(ns-unmap *ns* 'greeting)

(defmulti-joke greeting
  (fn[x] (x "language")))

;params is not used, so we could have used [_]
(defmethod-joke greeting "English" [params]
 "Hello!")

(defmethod-joke greeting "French" [params]
 "Bonjour!")

;;default handling
(defmethod-joke greeting :default [params]
  (str "I don't know the " (params "language") " language"))

;then can use this like this:
(def english-map {"id" "1", "language" "English"})
(def  french-map {"id" "2", "language" "French"})
(def spanish-map {"id" "3", "language" "Spanish"})

(println (greeting english-map))
;; => "Hello!"
(println (greeting french-map))
;; => "Bounjour!"
(println (greeting spanish-map))
;; =>  IllegalArgumentException: I don't know the Spanish language


;; Implementing factorial using multimethods Note that factorial-like function
;; is best implemented using `recur` which enables tail-call optimization to avoid
;; a stack overflow error. This is a only a demonstration of clojure's multimethod

;; identity form returns the same value passed
(ns-unmap *ns* 'factorial)

(defmulti-joke factorial identity)

(defmethod-joke factorial 0 [_]  1)
(defmethod-joke factorial :default [num]
    (* num (factorial (dec num))))

(println (factorial 0)) ; => 1
(println (factorial 1)) ; => 1
(println (factorial 3)) ; => 6
(println (factorial 7)) ; => 5040


;; defmulti/defmethods support variadic arguments and dispatch functions.

(def N #?(:clj Long
          :joker Int))

(ns-unmap *ns* 'bat)

(defmulti-joke bat
  (fn ([x y & xs]
       (mapv class (into [x y] xs)))))
(defmethod-joke bat [String String] [x y & xs]
  (str "str: " x " and " y))
(defmethod-joke bat [String String String] [x y & xs]
  (str "str: " x ", " y " and " (first xs)))
(defmethod-joke bat [String String String String] [x y & xs]
  (str "str: " x ", " y ", " (first xs) " and " (second xs)))
(defmethod-joke bat [N N] [x y & xs]
  (str "numbers: " x " and " y))

;; you call it like this...

(println (bat "mink" "stoat"))
;; => "str: mink and stoat"

(println (bat "bear" "skunk" "sloth"))
;; => "str: bear, skunk and sloth"

(println (bat "dog" "cat" "cow" "horse"))
;; => "str: dog, cat, cow and horse"

(println (bat 1 2))
;; => "numbers: 1 and 2"

(println (try (bat :hey :there)
              (catch #?(:clj Exception
                        :joker Error)
                  e
                (str "Caught this expected exception: " e))))
;; => IllegalArgumentException No method in multimethod 'bat' for dispatch value: [clojure.lang.Keyword clojure.lang.Keyword]  clojure.lang.MultiFn.getFn (MultiFn.java:156)

(defmethod-joke bat :default [x y & xs]
  (str "default: " x " then " y " and finally " xs))

(println (bat :hey :there :you))
;; => "default: :hey then :there and finally (:you)"


;; If you're REPLing you might want to re-define the defmulti dispatch function
;; (which defmulti won't allow you to do). For this you can use `ns-unmap`:

(ns-unmap *ns* 'x)

(defmulti-joke x (fn[_] :inc))
(defmethod-joke x :inc [y] (inc y))
(defmethod-joke x :dec [y] (dec y))
(println (x 0)) ;; => 1
(defmulti-joke x (fn[_] :dec)) ;; Can't redefine :(
(println (x 0)) ;; => 1 ;; STILL :(
(ns-unmap *ns* 'x) ;; unmap the var from the namespace
(defmulti-joke x (fn[_] :dec))
(defmethod-joke x :inc [y] (inc y))
(defmethod-joke x :dec [y] (dec y))
(println (x 0)) ;; => -1

;; So in your file while developing you'd put the ns-unmap to the top of the file


;; It's nice for multimethods to have arglists metadata so that calling `doc`
;; prints the arglist, instead of just the docstring. For example:
(use #?(:clj '[clojure.repl]
        :joker '[joker.repl]))

(ns-unmap *ns* 'f)

(defmulti-joke f "Great function" (fn [x] :blah))
(doc f)
;; => -------------------------
;; => user/f
;; =>   Great function

;; However, we can add `:arglists` metadata via a third (optional) argument to `defmulti` (`attr-map?` in the docstring for `defmulti`):

(ns-unmap *ns* 'g)

(defmulti-joke g "Better function" {:arglists '([x])} (fn [x] :blah))
(doc g)
;; => -------------------------
;; => user/f
;; => ([x])
;; =>   Better function


(ns-unmap *ns* 'compact)

(defmulti-joke compact map?)

(defmethod-joke compact true [map]
  (into {} (remove (comp nil? second) map)))

(defmethod-joke compact false [col]
  (remove nil? col))

;; Usage:

(println (compact [:foo 1 nil :bar]))
;; => (:foo 1 :bar)

(println (compact {:foo 1 :bar nil :baz "hello"}))
;; => {:foo 1, :baz "hello"}


;; This show how to do a wildcard match to a dispatch value:
;; (defmulti xyz (fn [x y] [x y]))

;; We don't care about the first argument:
;; (defmethod xyz [::default :b]
;;   [x y]
;;   :d-b)

;; We have to implement this manually:
;; (defmethod xyz :default
;;   [x y]
;;   (let [recover (get-method xyz [::default y])]
;;     ;; Prevent infinite loop:
;;     (if (and recover (not (= (get-method xyz :default) recover)))
;;       (do
;;         (println "Found a default")
;;         ;; Add the default to the internal cache:
;;         ;; Clojurescript will want (-add-method ...)
;;         (.addMethod ^MultiFn xyz [x y] recover)
;;         (recover ::default y))
;;       :default)))

;; (xyz nil :b) ;; => :d-b
;; only prints "Found a default" once!


;; Extremely simple example, dispatching on a single field of the input map.
;; Here we have a polymorphic map that looks like one of these two examples:

;;  {:name/type :split :name/first "Bob" :name/last "Dobbs"}
;;  {:name/type :full :name/full "Bob Dobbs"}

(ns-unmap *ns* 'full-name)

(defmulti-joke full-name :name/type)

(defmethod-joke full-name :full [name-data]
  (:name/full name-data))

(defmethod-joke full-name :split [name-data]
  (str (:name/first name-data) " " (:name/last name-data)))

(defmethod-joke full-name :default [_] "???")

(println (full-name {:name/type :full :name/full "Bob Dobbs"}))
;; => "Bob Dobbs"

(println (full-name {:name/type :split :name/first "Bob" :name/last "Dobbs"}))
;; => "Bob Dobbs"

(println (full-name {:name/type :oops :name/full "Bob Dobbs"}))
;; => "???"


;;polymorphism classic example

;;defmulti
(ns-unmap *ns* 'draw)

(defmulti-joke draw :shape)

;;defmethod
(defmethod-joke draw :square [geo-obj] (str "Drawing a " (:clr geo-obj) " square"))
(defmethod-joke draw :triangle [geo-obj] (str "Drawing a " (:clr geo-obj) " triangle"))

(defn square [color] {:shape :square :clr color})
(defn triangle [color] {:shape :triangle :clr color})

(println (draw (square "red"))) ; => "Drawing a red square"
(println (draw (triangle "green"))) ; => "Drawing a green triangle"


;;defmulti with dispatch function
(ns-unmap *ns* 'salary)

(defmulti-joke salary (fn[amount] (amount :t)))

;;defmethod provides a function implementation for a particular value
(defmethod-joke salary "com" [amount] (+ (:b amount) (/ (:b amount) 2)))
(defmethod-joke salary "bon" [amount] (+ (:b amount) 99))

(println (salary {:t "com" :b 1000})) ; => 1500
(println (salary {:t "bon" :b 1000})) ; => 1099

;;do these again, to catch errors relating to reuse of a shared resource across multimethods
(println (draw (square "red"))) ; => "Drawing a red square"
(println (draw (triangle "green"))) ; => "Drawing a green triangle"
