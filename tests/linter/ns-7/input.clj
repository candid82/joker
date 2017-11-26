(create-ns 'test.test)
(require 'joker.os)
(alias 'os 'joker.os)
(alias 't 'test.test)

(os/sh "ls")
(test.test/foo)
(t/bar)
