(defprotocol P
  (m [_this _a]))

(extend-type Object
  P
  (m [_this a]
    (pr-str a))
  (m1))
