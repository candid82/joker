(def f
  (fn [a]
    (def f1
      (fn [b]
        (def f11 (fn [] b))
        (+ a b)))
    (def f2 (fn [c] (+ a c)))
    (f1 10)
    (f2 20)
    (f11)))
