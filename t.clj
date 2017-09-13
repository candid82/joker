(defn test-method
  [])

(defrecord TestRecord [a b]
  TestProtocol
  (test-method [x y]
    (println x)
    (println x y a b c)
    (println a b))

  (test-method1 [x y]
    (println x)
    (println x y a b c d)
    (println a b)))

; (defn ->TestRecord
;   [a b]
;   (defn test-method
;     [x y]
;     (x y a b)))
