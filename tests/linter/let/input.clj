;; Should PASS

(let [a 1]
  a)

(let [{:keys [:t ::g]} {:t 1 :user/g 2}] [t g])

;; Should FAIL
(let [_ 1])
(let [] 1)
(let [[a & more 1] []] 1)
