(ns-unmap *ns* 'bat)

(def N #?(:clj Long
          :joker Int))

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
(defmethod-joke bat :default [x y & xs]
  (str "default:" x " then " y " and finally " xs))

;; you call it like this...

(println (bat "mink" "stoat"))
;; => "str: mink and stoat"

(println (bat "bear" "skunk" "sloth"))
;; => "str: bear, skunk and sloth"

(println (bat "dog" "cat" "cow" "horse"))
;; => "str: dog, cat, cow and horse"

(println (bat 1 2))
;; => "numbers: 1 and 2"

(println (bat :hey :there))
;; => IllegalArgumentException No method in multimethod 'bat' for dispatch value: [clojure.lang.Keyword clojure.lang.Keyword]  clojure.lang.MultiFn.getFn (MultiFn.java:156)


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
