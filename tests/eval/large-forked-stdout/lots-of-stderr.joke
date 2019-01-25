(println "this is stdout; just one line.")
(doseq [n (range 10000)  ; Should be enough to block writing stderr until parent reads it.
        ]
  (println-err (format "A lazy dog knows %d tricks." n)))
