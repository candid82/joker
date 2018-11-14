;; Should PASS

(if 1 2 3)
(if-let [a 1] a 2)

;; Should FAIL

(if 1 2)
(if-let [a 1] a)
