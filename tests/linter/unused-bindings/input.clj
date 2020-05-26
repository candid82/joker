;; Should PASS
(let [a 1] a)
(let [_ 1] 1)
(loop [[a & b] [1 2]]
  (if a
    (recur b)
    1))

;; Should FAIL

(let [a 1 b 2] b)
(let [a 1]
  (let [a 2]
    a))

(loop [[a & b] [1 2]]
  (if 1
    (recur b)
    1))

(let [{:keys [a b]} {}] (println a))

(let [_ 1 _ 2 a 1 a 2] a)

(defn f [{}] 1)
(defn f1 [[]] 1)
