(defprotocol P)

(extend-protocol P
  Integer
  (m1 [a] nil)
  (m2))
