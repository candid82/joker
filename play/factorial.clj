(defn fac
  [n]
  (if (zero? n)
    1
    (* n (fac (- n 1)))))

(defn fac1
  [n]
  (loop [res 1 x n]
    (if (zero? x)
      res
      (recur (* res x) (- x 1)))))

; 1 1 2 3 5 8 ...
; (defn fib
;   [n]
;   ())
