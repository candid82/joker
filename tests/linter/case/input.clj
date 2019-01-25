;; Should PASS
(case 1 ("F") false true)

;; Should FAIL
(case 1 (1 2) false (2 4) true 4 false)
