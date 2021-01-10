;; Should PASS
(doseq [_ (range 10)] 1)

;; Should FAIL
(doseq [_ (range 10)])
