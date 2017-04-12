;; Should PASS

(loop [a 1]
  (if (< a 10)
    (recur (inc a))
    a))

;; Don't currently check for recursion points due
;; to potential false positives in macros / defmethods
(let [a 1]
  (if (< a 10)
    (recur)
    a))

;; Should FAIL

(loop [a 1]
  (if (< a 10)
    (recur)
    a))
