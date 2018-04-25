;; Should PASS

(require '[test.ns1 :as ns1])

(let [a 1]
  a)

(let [{:keys [:t ::g]} {:t 1 :user/g 2}] [t g])
(let [{::ns1/keys [a]} {:test.ns1/a 1}] a)

;; Should FAIL
(let [_ 1])
(let [] 1)
(let [foo/bar 2] bar)
(let [[a & more 1] []] 1)
