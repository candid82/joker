(defmacro check-native-core []
  ;; Macro execution must not use the linter's public transient stubs.
  (let [checks [(= [2 3] (mapv inc [1 2]))
                (= [[1 3 5 7] [2 4 6 8]]
                   (mapv vector [1 2] [3 4] [5 6] [7 8]))
                (= [2] (filterv even? [1 2]))
                (= {:a 2 :b 1} (frequencies [:a :b :a]))
                (= {:a 1 :b 2} (zipmap [:a :b] [1 2]))
                (= {:a 1} (select-keys {:a 1 :b 2} [:a]))
                (= [0 1 2] (vec (range 3)))
                (= [:x :x] (vec (repeat 2 :x)))
                (= [false] (vec (keep identity [nil false])))
                (= [[0 :a] [1 :b]] (vec (map-indexed vector [:a :b])))
                (= [false] (vec (keep-indexed (fn [_ x] x) [nil false])))
                (= [2 3] (vec (drop 2 [0 1 2 3])))
                (= [0 1] (vec (take-while #(< % 2) [0 1 2 3])))
                (= [2 3] (vec (drop-while #(< % 2) [0 1 2 3])))
                (= [1 2] (vec (distinct [1 1 2])))
                (= [[1 3] [2 4]] (vec (map vector [1 2] [3 4])))
                (= 1 (get-in {:a {:b 1}} [:a :b]))
                (= {:a {:b 1}} (assoc-in {} [:a :b] 1))
                (= {:a {:b 3}} (update-in {:a {:b 1}} [:a :b] + 2))]]
    (when-not (every? true? checks)
      (throw (ex-info "Native core macro execution differs" {:checks checks})))))

(check-native-core)
