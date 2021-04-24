<img src="https://user-images.githubusercontent.com/882970/48048842-a0224080-e151-11e8-8855-642cf5ef3fdd.png" width="117px"/>

# Floating-point Constants and the BigFloat Type

Floating-point constants suffixed with `M` become type `BigFloat`. Compare `joker.math/e`, a truncated version of Go's `math.E` due to forcing it into a `Double`, to the original value with Clojure's `M` suffix:

```
user=> joker.math/e
2.718281828459045
user=> 2.71828182845904523536028747135266249775724709369995957496696763M
2.71828182845904523536028747135266249775724709369995957496696763M
user=>
```

Further, `BigFloat`s created from strings (when a constant such as `1.3M` is parsed by Joker) are given a _minimum_ precision of 53 (the same precision as a `Double`, aka `float64`) and a _maximum_ precision based on the number of digits and the number of bits each digit represents (3.33 for decimal; 1, 3, or 4 for binary, octal, and hex).

The `joker.math/precision` function returns the precision for a `BigFloat` type, though it supports a few others (e.g. `(joker.math/precision 1)` returns `63N` on 64-bit systems, `31N` on 32-bit).

This combines to produce a fairly useful set of default behaviors:

```
user=> (joker.math/precision 2.71828182845904523536028747135266249775724709369995957496696763M)
208N
user=> (def c1 1.3M)
#'user/c1
user=> (def c2 0000000000001.3000000000000M)
#'user/c2
user=> (joker.math/precision c1)
53
user=> (joker.math/precision c2)
86
user=> (= c1 c2)
false
user=> (+ c1 c2)
2.600000000000000044408921M
user=>
```

Note the inequality of the two values: when `c1` is binary-extended to match the precision of `c2`, it represents a slightly different value than does `c1`, as confirmed when adding the two values. This effect is not observed when using binary-based (binary, octal, or hexadecimal) encoding, since they encode the mantissa and thus the value precisely, which base-10 encoding (shown above) does not, due to the underlying representation using binary (rather than decimal) digits. E.g.:

```
user=> (= 0x1.fM 0x1.fM)
true
user=> (= 0x1.fM 0x1.f00000000000000000000000000M)
true
user=>
```

Finally, `joker.math/set-precision` can be used to make a copy of a `BigFloat` with the specified precision, _a la_ Go's `math/big.(*Float).SetPrec()` method:

```
user=> (def big-e 2.71828182845904523536028747135266249775724709369995957496696763M)
#'user/big-e
user=> (joker.math/precision big-e)
208N
user=> (def truncated-e (joker.math/set-precision 53 big-e))
#'user/truncated-e
user=> truncated-e
2.718281828459045M
user=> (joker.math/precision truncated-e)
53N
user=> (joker.math/precision joker.math/e)
53N
user=>
```
