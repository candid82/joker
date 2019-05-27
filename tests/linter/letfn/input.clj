(letfn [(neven? [n] (if (zero? n) true (nodd? (dec n))))
        (nodd? [n] (if (zero? n) false (neven? (dec n))))]
  (neven? 10))


(letfn)
