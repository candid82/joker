(defprotocol P1)
(defprotocol P2)

(reify
  P1
  (m1 [a])
  (m2 [b])

  P2
  (m3 [c])
  (m4))
