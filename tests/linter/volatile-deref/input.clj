(ns volatile-deref)

(let [v (volatile! 0)]
  @v)

(defn factorial [n]
  (let [acc (volatile! 1)
        i (volatile! n)]
    (while (pos? @i)
      (vswap! acc *' @i)
      (vswap! i dec))
    @acc))
