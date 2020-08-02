(do
  3
  (do
    1
    2))

(let [a 1]
  (do
    1
    a))

(defn f
  []
  (do 1 2))

(if-let [a 1]
  a
  2)

(if-let [a 1]
  (do 3 a)
  2)

(when-let [a 1]
  (do 1 a))

#(do (println 1) (println 2))

#(do (println 1))
