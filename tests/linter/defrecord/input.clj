(ns test.test
  (:require [test.prococol1 :refer [TestProtocol1]]))

(defprotocol TestProtocol)

(defn test-method
  [] nil)

(defrecord TestRecord [a b]
  TestProtocol
  (test-method [x y]
    (println x)
    (println x y a b c)
    (println a b))

  TestProtocol1
  (test-method1 [x y]
    (println x)
    (println x y a b c d)
    (println a b))

  TestProtocol2
  (test-method2 [x y]
    (println x)))

(instance? TestRecord 1)
(map->TestRecord 1)
(->TestRecord 1 2)

(map->TestRecord 1 2)
(->TestRecord 1)

(defrecord ^:private TestRecord1 [])
(deftype TestType [])

(->TestType)
