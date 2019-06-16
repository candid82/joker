(if 1 2 3)

(if 1 2 )

(if 1)

(if)

(fn [a b] 1 2)

(fn a [a b] 1 2)

(let [a 1 b 2] 1 2)

(let [] 1)

(let [a 1] 1 2)

(let [a] 1 2)

(letfn [(neven? [n] (if (zero? n) true (nodd? (dec n))))
        (nodd? [n] (if (zero? n) false (neven? (dec n))))]
  (neven? 10))
