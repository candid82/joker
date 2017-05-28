;; Should PASS
(try (catch Error e (loop [a 1] (recur (inc a)))))
