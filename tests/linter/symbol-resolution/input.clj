(ns tests.symbols-lint
  (:require [tests.ext1]
            [tests.ext2 :as ext2]
            [tests.ext3 :as ext3 :refer [e3]]
            [spec :as s]
            [clojure.test :refer [testing is]])
  (:import [java.security Security]))

;; Should PASS

(defmacro test-macro [_x] nil)

(def f [])

(f 1)
(tests.ext1/f 1)
(e3 1)
(ext2/g 4)
(defprotocol Person)
(java.security.Security/tt)
(Security/hh)
(Integer/parseInt 10)
(java.lang.Integer/parseInt 10)
(instance? Integer 10)
(instance? java.lang.Integer 10)
(test-macro uuu)
(defrecord u5 [])
(clojure.test/deftest u7)
#'s/v
#'e3
(case 1 r 2 1 3)
uuu
(is (thrown? c body))

;; Should FAIL

(s/def :t hh)
(f jj/u1)
(f u8)
(f1 1)
(tt/g 3)
(test-macro (f u2))
(test-macro (load u3))
(test-macro (ext2/f u4))
(testing u9)
(defmethod u6 1 [] nil)
#'g
#'ns/gg
(joker.string/split "1" #"2")
(joker.os/env)
(joker.time/sleep 10)
(joker.json/read-string "")
(joker.base64/decode-string "")
(pprint 1)
(pr-err 1)
(prn-err 1)
(print-err 1)
(println-err 1)
(case 1 r 2 1 h)
