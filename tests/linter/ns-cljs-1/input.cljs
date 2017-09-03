(ns hello.world
  (:require [om.next :as om :refer-macros [defui]]
            [om.dom :as dom]))

(defui HelloWorld
  Object
  (render [this]
          (dom/div nil "Hello, World!")))
