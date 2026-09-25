(ns volatile-deref-cljs)

(let [v (volatile! 0)]
  @v)
