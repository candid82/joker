(ns test
  (:require [clojure.string :refer [starts-with?] :rename {starts-with? begins-with?}]))

(begins-with? "test" "t")
(starts-with? "test" "t")
