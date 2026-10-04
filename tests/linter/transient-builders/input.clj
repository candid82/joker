(require '[joker.set :as sets])

;; These builders must execute during macro expansion despite the lint-mode
;; stubs for public transient operations.
(defmacro check-builders []
  (let [m (zipmap [0 1] [10 11])
        results [(select-keys m [1])
                 (frequencies [0 0 1])
                 (set [0 0 1])
                 (update-keys m inc)
                 (update-vals m inc)
                 ((juxt inc inc inc inc) 0)
                 (sets/union #{0} #{1})
                 (sets/intersection #{0 1} #{1})
                 (sets/difference #{0 1} #{1})
                 (sets/select odd? #{0 1})
                 (sets/rename-keys m {0 1 1 0})
                 (sets/map-invert m)
                 (sets/index [{:id 0} {:id 0}] [:id])
                 (sets/join #{{:id 0}} #{{:id 0 :value 10}})]]
    (if (= [{1 11} {0 2 1 1} #{0 1} {1 10 2 11} {0 11 1 12}
            [1 1 1 1] #{0 1} #{1} #{0} #{1} {1 10 0 11} {10 0 11 1}
            {{:id 0} #{{:id 0}}} #{{:id 0 :value 10}}]
           results)
      nil
      (throw (ex-info "Collection builders failed during macro expansion" {})))))

(check-builders)
