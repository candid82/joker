(defprotocol P
  (m [this a]))

(extend-type Object
  P
  (m [this a]
    (pr-str a))
  (m1 ))
