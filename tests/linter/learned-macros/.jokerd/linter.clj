(ns rum.core)

(defmacro defc [& params]
	(let [[name _ _ arg-vector & body] params]
   `(defn ~name ~arg-vector ~@body)))


(in-ns 'user)
