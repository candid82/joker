(ns typed-core-returns-cljs)

(+ (unchecked-add-int 1 2)
   (unchecked-int 3)
   (long 4))

(subs (munge "a-b") 0)
(keys (sorted-map :a 1))
(disj (sorted-set :a) :a)
(first (replicate 1 :x))
@(ensure-reduced 1)
