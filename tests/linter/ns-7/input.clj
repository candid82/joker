(create-ns 'test.test)
(alias 'os 'joker.os)
(alias 't 'test.test)

(os/sh "ls")
(test.test/foo)
(t/bar)
