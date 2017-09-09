(defn f1 [])
(defn f2 [x])
(def v1 1)
(def k :t)
;; Should PASS

(spit "test" "test" :append true)
(map identity)
(f1)
(f2 1)
(k {})
((fn []))
(#{} 1)
({1 2} 1)

;; Should FAIL
(map)
(*assert*)
(f1 1)
(f2)
(v1)
(1)
("")
((fn []) 1)
(joker.string/split "asdf")

