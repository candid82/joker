;; Should PASS

(def v1)
(def v2 1)

;; Should FAIL

(def ^:private v3)
(def ^:private v4 1)
(defmacro ^:private m [x] x)
(defn ^:private f [] 1)
