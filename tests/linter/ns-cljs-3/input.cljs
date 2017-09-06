(ns example.macros
  (:require-macros [cljs.core.async.macros :as async])
  (:require [cljs.core.async :as async]
            [left-pad :as lp]))

(async/t)
(left-pad)
(lp 1)
(t)
