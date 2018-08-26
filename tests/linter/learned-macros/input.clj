(require '[rum.core :as rum])

(rum/defc Foo < rum/static [{:keys [bar/foo]}] [:div (str foo)])
