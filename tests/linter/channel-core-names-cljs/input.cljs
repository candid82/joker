;; Joker-only channel operations must not leak into cljs.core/user.
(offer! nil :value)
(poll! nil)
