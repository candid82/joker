(require '[joker.hiccup :as hiccup])

;; Rendering at macro-expansion time must not depend on lint-mode stubs for
;; public transient operations. Cover the zero, one, and sorted many cases.
(defmacro check-render []
  (if (= ["<div></div>" "<div a=\"1\"></div>" "<div a=\"1\" b=\"2\"></div>"]
         [(hiccup/html [:div {:a nil :b false}])
          (hiccup/html [:div {:a 1}])
          (hiccup/html [:div {:b 2 :a 1}])])
    nil
    (throw (ex-info "Macro-time Hiccup rendering failed" {}))))

(check-render)
