(require '[clojure.walk :as walk])

(defmacro check-native-walk []
  ;; These run during macro expansion, when public transient functions are
  ;; replaced by linter stubs. Native reconstruction must still work.
  (let [checks [(= [2 {:a 3}] (walk/postwalk #(if (number? %) (inc %) %) [1 {:a 2}]))
                (= [:c 2] (walk/prewalk-replace {:a [:b 1] :b :c 1 2} :a))
                (= {:a {:b 1}} (walk/keywordize-keys {"a" {"b" 1}}))
                (= {"a" {"b" 1}} (walk/stringify-keys {:a {:b 1}}))
                (= '(10 20) (walk/walk {:a 10 :b 20} identity '(:a :b)))]]
    (when-not (every? true? checks)
      (throw (ex-info "Native macro-time walking differs" {:checks checks})))))

(check-native-walk)
