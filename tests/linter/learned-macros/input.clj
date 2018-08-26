(require '[rum.core :as rum])
(require '[rum.core1 :as rum1])

(rum/defc Foo < rum/static [{:keys [bar/foo]}] [:div (str foo)])
