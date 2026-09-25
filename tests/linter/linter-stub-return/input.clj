(ns linter-stub-return)

(let [a (byte-array 1)]
  (inc (aset-byte a 0 1)))
