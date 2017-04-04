;; Should PASS
#?(:clj 1)
#?@(:cljs 3)
(def regexp #?(:clj re-pattern :cljs js/XRegExp))


;; Should FAIL
#?(:cljs)
#?(:cljs (let [] 1) :default (let [] 1))
#?@(:cljs 3 :clj (let [_ 1]))
#?(:clj 234ewr :cljs 2)

