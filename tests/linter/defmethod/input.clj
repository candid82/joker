(defmulti m1)

;; Should FAIL
(defmethod m1 :v
  [_a]
  b)

(defmethod m2 :v [] nil)

(defmethod m3)
