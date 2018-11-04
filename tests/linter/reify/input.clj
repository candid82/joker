(defprotocol P1)
(defprotocol P2)

(reify
  P1
  (m1 [a] nil)
  (m2 [b] nil)

  P2
  (m3 [c] nil)
  (m4))
