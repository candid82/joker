(ns rum.core)

(defmacro defc [& params]
	(let [[name _ _ arg-vector & body] params]
   `(defn ~name ~arg-vector ~@body)))

(ns rum.core1)

(defmacro defc1 [& params]
	(let [[name _ _ arg-vector & body] params]
   `(defn ~name ~arg-vector ~@body)))

(ns rum.core2)

(defmacro defc2 [& params]
	(let [[name _ _ arg-vector & body] params]
   `(defn ~name ~arg-vector ~@body)))


(in-ns 'user)
