(ns test.ns)

(defprotocol ITest
  (x [this]))

(extend-protocol ITest
  datomic.db.Db
  (x [this] (println "do something")))
