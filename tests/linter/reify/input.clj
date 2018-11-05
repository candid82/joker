(defprotocol P1)
(defprotocol P2)

(reify
  P1
  (m1 [_a] nil)
  (m2 [_b] nil)

  P2
  (m3 [_c] nil)
  (m4))
