;; Should PASS

(seq {})
(seq [])
(seq #{})

;; Should FAIL

(seq 1)
(seq first)
(def v 1)
(seq v)
(def ^Int v1 (if 1 2 3))
(seq v1)
(def ^:dynamic dv 1)
(seq dv)
