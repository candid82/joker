(defprotocol P)

(extend-protocol P
  Integer
  (m1 [_a] nil)
  (m2))
