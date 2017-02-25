;; Should PASS
#?(:clj 1)
#?@(:cljs 3)
(def regexp #?(:clj re-pattern :cljs js/XRegExp))
#?@(:clj (let [a 1]) :cljs 3)


;; Should FAIL
#?(:cljs (let [] 1) :default (let [] 1))
#?(:clj 234ewr :cljs 2)

