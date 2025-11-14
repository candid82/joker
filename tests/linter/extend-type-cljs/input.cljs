(defprotocol P
  (m [_this _a]))

(extend-type object
  P
  (m [_this a]
    (pr-str a))
  (m1))

(extend-type object IUUID)
