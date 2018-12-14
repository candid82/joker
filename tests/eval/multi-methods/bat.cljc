(ns-unmap *ns* 'bat)

(def N #?(:clj Number
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
;; (defmethod-joke bat :default [x y & xs]
;;   (str "default:" x " then " y " and finally " xs))

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
