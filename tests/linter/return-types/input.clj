(defn ^Map parse-address [s]
  {})

(defn ^String email-from [from]
  (parse-address from))

(defn ^Map map-literal []
  {})

(defn ^Seqable vector-as-seqable []
  [])

(defn ^"Number|Char" union-number []
  1)

(defn ^"Number|Char" union-char []
  \a)

(defn ^"Number|Char" union-bad []
  "x")

(defn ^String string-from-union []
  (union-number))

