(ns my.test (:require [com.rpl.specter :refer :all]))

(transform [MAP-VALS ALL MAP-VALS even?] inc [])
