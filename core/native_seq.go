package core

import "math"

// Native lazy bodies use the consumer's ambient execution. They retain only
// values and callbacks, never a creator's VM or suspended execution context.
// Keeping LazySeq/ConsSeq also preserves metadata, realized? and retry behavior.
func nativeLazy(body ProcFn) *LazySeq {
	return &LazySeq{fn: Proc{Fn: body}}
}

func nativeMapMulti(callback Object, sources []Object) Seq {
	return nativeMapMultiState(newNativeCallback(callback, len(sources)), sources)
}

func nativeMapMultiState(callback *nativeCallback, sources []Object) Seq {
	if len(sources) > 3 {
		rows := nativeMapRows(&ArraySeq{arr: sources}, len(sources))
		return NewMapSeq(Proc{Fn: func(args []Object) Object {
			cursor := newSeqCursor(EnsureArgIsSeqable(args, 0).Seq())
			for i := range callback.args {
				cursor.hasNext()
				callback.args[i] = cursor.first()
				cursor.advance()
			}
			// Realize the empty row tail, as apply's ToSlice does.
			cursor.hasNext()
			return callback.call()
		}}, rows)
	}
	return nativeLazy(func(_ []Object) Object {
		var seqs [3]Seq
		present := true
		// The two/three-source arities realize ALL sources before testing
		// exhaustion, unlike the variadic arity's short-circuiting step.
		for i, source := range sources {
			seqs[i] = EnsureObjectIsSeqable(source, "").Seq()
			if seqs[i].IsEmpty() {
				present = false
			}
		}
		if !present {
			return EmptyList
		}
		for i := range sources {
			callback.args[i] = seqs[i].First()
		}
		value := callback.call()
		rest := make([]Object, len(sources))
		for i := range sources {
			rest[i] = seqs[i].Rest()
		}
		return NewConsSeq(value, nativeMapMultiState(callback, rest))
	})
}

// Variadic map's row preparation short-circuits at the first empty input, and
// delays source Rest calls until the next row, after the current callback.
func nativeMapRows(sources Seqable, width int) Seq {
	return nativeLazy(func(_ []Object) Object {
		seqs := make([]Object, 0, width)
		for cursor := newSeqCursor(sources.Seq()); cursor.hasNext(); cursor.advance() {
			seq := EnsureObjectIsSeqable(cursor.first(), "").Seq()
			if seq.IsEmpty() {
				return EmptyList
			}
			seqs = append(seqs, seq)
		}
		row := NewMapSeq(Proc{Fn: procFirst}, &ArraySeq{arr: seqs})
		// Cache each Rest independently. If a later source throws, retrying
		// row preparation must not call an earlier source's Rest again.
		rest := NewMapSeq(Proc{Fn: procRest}, &ArraySeq{arr: seqs})
		return NewConsSeq(row, nativeMapRows(rest, width))
	})
}

var procMapMulti = func(args []Object) Object {
	CheckArity(args, 3, math.MaxInt)
	return nativeMapMulti(args[0], append([]Object(nil), args[1:]...))
}

func nativeRepeat(value Object) Seq {
	return nativeLazy(func(_ []Object) Object {
		return NewConsSeq(value, nativeRepeat(value))
	})
}

var procRepeat = func(args []Object) Object {
	CheckArity(args, 1, 1)
	return nativeRepeat(args[0])
}

func nativeRange(start, end, step Object) Seq {
	return nativeLazy(func(_ []Object) Object {
		n := EnsureObjectIsNumber(step, "")
		ops := GetOps(n)
		equal := start.Equals(end)
		present := false
		switch {
		case ops.IsZero(n) || equal:
			present = !equal
		case ops.Gt(n, Int{I: 0}):
			x, y := EnsureObjectIsNumber(start, ""), EnsureObjectIsNumber(end, "")
			present = GetOps(x).Combine(GetOps(y)).Lt(x, y)
		case ops.Lt(n, Int{I: 0}):
			x, y := EnsureObjectIsNumber(start, ""), EnsureObjectIsNumber(end, "")
			present = GetOps(x).Combine(GetOps(y)).Gt(x, y)
		default:
			// A NaN step selects no comparator in range's conditional.
			EnsureObjectIsCallable(NIL, "")
		}
		if !present {
			return EmptyList
		}
		// Compute the next start while realizing THIS node, as (+ start
		// step) is evaluated before constructing the recursive lazy call.
		x := EnsureObjectIsNumber(start, "")
		next := GetOps(x).Combine(GetOps(n)).Add(x, n)
		return NewConsSeq(start, nativeRange(next, end, n))
	})
}

func nativeInfiniteRangeRest(current Number) Seq {
	return nativeLazy(func(_ []Object) Object {
		next := GetOps(current).Combine(INT_OPS).Add(current, Int{I: 1})
		return NewConsSeq(next, nativeInfiniteRangeRest(next))
	})
}

var procRange = func(args []Object) Object {
	CheckArity(args, 0, 3)
	switch len(args) {
	case 0:
		// (range) is iterate's eager first cons, not a lazy root.
		return NewConsSeq(boxInt(0), nativeInfiniteRangeRest(boxInt(0)))
	case 1:
		return nativeRange(boxInt(0), args[0], boxInt(1))
	case 2:
		return nativeRange(args[0], args[1], boxInt(1))
	default:
		return nativeRange(args[0], args[1], args[2])
	}
}

func nativeKeep(callback *nativeCallback, source Object, index int, indexed, skipNil bool) Seq {
	return nativeLazy(func(_ []Object) Object {
		seq := EnsureObjectIsSeqable(source, "").Seq()
		if seq.IsEmpty() {
			return EmptyList
		}
		if indexed {
			callback.args[0], callback.args[1] = boxInt(index), seq.First()
		} else {
			callback.args[0] = seq.First()
		}
		value := callback.call()
		rest := nativeKeep(callback, seq.Rest(), index+1, indexed, skipNil)
		if _, nilValue := value.(Nil); skipNil && nilValue {
			// Returning the next lazy node (rather than scanning here)
			// preserves realized? when a later skipped callback throws.
			return rest
		}
		return NewConsSeq(value, rest)
	})
}

var procKeep = func(args []Object) Object {
	CheckArity(args, 2, 2)
	return nativeKeep(newNativeCallback(args[0], 1), args[1], 0, false, true)
}

var procMapIndexed = func(args []Object) Object {
	CheckArity(args, 2, 2)
	return nativeKeep(newNativeCallback(args[0], 2), args[1], 0, true, false)
}

var procKeepIndexed = func(args []Object) Object {
	CheckArity(args, 2, 2)
	return nativeKeep(newNativeCallback(args[0], 2), args[1], 0, true, true)
}

func nativeTakeWhile(callback *nativeCallback, source Object) Seq {
	return nativeLazy(func(_ []Object) Object {
		seq := EnsureObjectIsSeqable(source, "").Seq()
		if seq.IsEmpty() {
			return EmptyList
		}
		callback.args[0] = seq.First()
		if !ToBool(callback.call()) {
			return EmptyList
		}
		return NewConsSeq(seq.First(), nativeTakeWhile(callback, seq.Rest()))
	})
}

var procTakeWhile = func(args []Object) Object {
	CheckArity(args, 2, 2)
	return nativeTakeWhile(newNativeCallback(args[0], 1), args[1])
}

var procDrop = func(args []Object) Object {
	CheckArity(args, 2, 2)
	count, source := args[0], args[1]
	return nativeLazy(func(_ []Object) Object {
		seq := EnsureObjectIsSeqable(source, "").Seq()
		cursor := newSeqCursor(seq)
		present := cursor.hasNext()
		n := EnsureObjectIsNumber(count, "")
		advanced := false
		for GetOps(n).Gt(n, Int{I: 0}) && present {
			n = GetOps(n).Combine(INT_OPS).Subtract(n, Int{I: 1})
			cursor.advance()
			present = cursor.hasNext()
			advanced = true
		}
		if !present {
			return EmptyList
		}
		if !advanced {
			return seq
		}
		return cursor.rest()
	})
}

var procDropWhile = func(args []Object) Object {
	CheckArity(args, 2, 2)
	callback, source := newNativeCallback(args[0], 1), args[1]
	return nativeLazy(func(_ []Object) Object {
		seq := EnsureObjectIsSeqable(source, "").Seq()
		cursor := newSeqCursor(seq)
		advanced := false
		for cursor.hasNext() {
			callback.args[0] = cursor.first()
			if !ToBool(callback.call()) {
				if !advanced {
					return seq
				}
				return cursor.rest()
			}
			cursor.advance()
			advanced = true
		}
		return EmptyList
	})
}

func nativeDistinct(source Object, seen *MapSet) Seq {
	return nativeLazy(func(_ []Object) Object {
		cursor := newSeqCursor(EnsureObjectIsSeqable(source, "").Seq())
		for cursor.hasNext() {
			value := cursor.first()
			if found, _ := seen.Get(value); !found {
				cursor.advance()
				// Persistent state is essential: with-meta may clone an
				// unrealized node, and either clone can be consumed first.
				return NewConsSeq(value, nativeDistinct(cursor.rest(), seen.Conj(value).(*MapSet)))
			}
			cursor.advance()
		}
		return EmptyList
	})
}

var procDistinct = func(args []Object) Object {
	CheckArity(args, 1, 1)
	return nativeDistinct(args[0], EmptySet())
}
