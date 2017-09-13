;; Should PASS

[{:db/fn #db/fn  {:lang "java"
                  :params [db e doc]
                  :code "return list(list(\":db/add\", e, \":db/doc\", doc))"}}]

#inst 1
#uuid 2

;; Should FAIL

#t 4
#g [a]
