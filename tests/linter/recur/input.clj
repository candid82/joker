;; Should PASS
(try 1 (catch Error e (loop [a 1] (recur (inc a)))))

;; Should FAIL
(loop [a 1]
  (recur (inc a) nil))
