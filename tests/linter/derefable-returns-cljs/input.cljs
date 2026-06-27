(ns derefable-returns-cljs)

@(reduced 1)
@(add-watch (atom 1) :watcher (fn [_key _ref _old _new] nil))
@(remove-watch (atom 1) :watcher)
