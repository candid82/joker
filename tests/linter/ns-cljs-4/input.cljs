(ns test
  (:require ["m1" :as ns1]
            ["m2/t"]
            ["m3/t" :as ns3]
            [ns1]
            [m1]))

(ns3/f)
