(ns
  ^{:go-imports []
    :doc "Implements encoding and decoding of JSON as defined in RFC 4627."}
  json)

(defn read-string
  "Parses the JSON-encoded data and return the result as a Joker value.
  Optional opts map may have the following keys:
  :keywords? - if true, JSON keys will be converted from strings to keywords."
  {:added "1.0"
  :go {1 "readString(s, nil)"
       2 "readString(s, opts)"}}
  ([^String s])
  ([^String s ^Map opts]))

(defn write-string
  "Returns the JSON encoding of v."
  {:added "1.0"
  :go "writeString(v)"}
  [^Object v])


