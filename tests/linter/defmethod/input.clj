(defmulti m1)

;; Should FAIL
(defmethod m1 :v
  [a]
  b)

(defmethod m2 :v [])

(defmethod m3)
