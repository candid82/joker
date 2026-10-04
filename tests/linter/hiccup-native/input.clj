(require '[joker.hiccup :as hiccup])

(defmacro check-native-render []
  (if (= "<div class=\"a b\"><span>raw</span></div>"
         (hiccup/html [:div.a {:class "b"} (hiccup/raw-string "<span>raw</span>")]))
    nil
    (throw (ex-info "Native macro-time Hiccup rendering differs" {}))))

(check-native-render)
(hiccup/html)
(hiccup/html {:mode :xml} [:br])
