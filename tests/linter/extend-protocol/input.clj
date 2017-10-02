(defprotocol P)

(extend-protocol P
  Integer
  (m1 [a])
  (m2))
