;; Joker-only channel operations must not leak into clojure.core/user.
(offer! nil :value)
(poll! nil)
