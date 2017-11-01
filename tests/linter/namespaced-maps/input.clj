(alias 's 'joker.string)
;; Should PASS
#:t{:g 1}
#::{:g 1}
#:t{:_/g 1}
#:t{:h/g 1}
#::s{:g 1}

;; Should FAIL
#::{g 1}
