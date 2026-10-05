package core

import "math"

// Keep the Reduce protocol: host collections can implement it without Seqable.
// The native step receives the same private accumulator as a Joker reduction.
func reduceNative(source Object, step Proc, initial Object) Object {
	if reducer, ok := source.(Reduce); ok {
		return reducer.reduceInit(step, initial)
	}
	return seqReduceInit(EnsureObjectIsSeqable(source, "").Seq(), step, initial)
}

var procMapv = func(args []Object) Object {
	CheckArity(args, 2, math.MaxInt)
	if len(args) > 2 {
		return mapvMultiple(args[0], args[1:])
	}
	callback := newNativeCallback(args[0], 1)
	step := Proc{Fn: func(pair []Object) Object {
		callback.args[0] = pair[1]
		value := callback.call()
		return EnsureObjectIsTransientCollection(pair[0], "").ConjBang(value)
	}}
	builder := EmptyArrayVector().AsTransient()
	return EnsureObjectIsTransientCollection(reduceNative(args[1], step, builder), "").Persistent()
}

// Eager multi-source mapping can avoid allocating a lazy row/tail per item.
// Preserve map's arity-dependent realization order and own callback arguments.
func mapvMultiple(value Object, sources []Object) Object {
	callback := newNativeCallback(value, len(sources))
	builder := EmptyArrayVector().AsTransient()
	cursors := make([]seqCursor, len(sources))
	present := true
	for i, source := range sources {
		cursors[i] = newSeqCursor(EnsureObjectIsSeqable(source, "").Seq())
		if !cursors[i].hasNext() {
			if len(sources) > 3 {
				return builder.Persistent()
			}
			present = false
		}
	}
	for present {
		for i := range cursors {
			callback.args[i] = cursors[i].first()
		}
		builder = builder.ConjBang(callback.call())
		if len(sources) > 3 {
			for i := range cursors {
				cursors[i].advance()
				if !cursors[i].hasNext() {
					return builder.Persistent()
				}
			}
		} else {
			for i := range cursors {
				cursors[i].advance()
			}
			for i := range cursors {
				if !cursors[i].hasNext() {
					present = false
				}
			}
		}
	}
	return builder.Persistent()
}

var procFilterv = func(args []Object) Object {
	CheckArity(args, 2, 2)
	callback := newNativeCallback(args[0], 1)
	step := Proc{Fn: func(pair []Object) Object {
		callback.args[0] = pair[1]
		if ToBool(callback.call()) {
			return EnsureObjectIsTransientCollection(pair[0], "").ConjBang(pair[1])
		}
		return pair[0]
	}}
	return EnsureObjectIsTransientCollection(reduceNative(args[1], step, EmptyArrayVector().AsTransient()), "").Persistent()
}

var procFrequencies = func(args []Object) Object {
	CheckArity(args, 1, 1)
	step := Proc{Fn: func(pair []Object) Object {
		count := Number(boxInt(1))
		if counts, ok := pair[0].(Gettable); ok {
			if found, value := counts.Get(pair[1]); found {
				if integer, ok := value.(Int); ok {
					count = boxInt(integer.I + 1)
				} else {
					n := EnsureObjectIsNumber(value, "")
					count = GetOps(n).Combine(INT_OPS).Add(n, Int{I: 1})
				}
			}
		}
		return EnsureObjectIsTransientAssociative(pair[0], "").AssocBang(pair[1], count)
	}}
	return EnsureObjectIsTransientCollection(reduceNative(args[0], step, EmptyArrayMap().AsTransient()), "").Persistent()
}

var procZipmap = func(args []Object) Object {
	CheckArity(args, 2, 2)
	keys := newSeqCursor(EnsureArgIsSeqable(args, 0).Seq())
	keysPresent := keys.hasNext()
	values := newSeqCursor(EnsureArgIsSeqable(args, 1).Seq())
	valuesPresent := values.hasNext()
	result := EmptyArrayMap().AsTransient().(TransientMapCollection)
	for keysPresent && valuesPresent {
		result = result.AssocBang(keys.first(), values.first()).(TransientMapCollection)
		// Both next calls are evaluated, including their empty checks. Do
		// not short-circuit the value source when the key source ends.
		keys.advance()
		keysPresent = keys.hasNext()
		values.advance()
		valuesPresent = values.hasNext()
	}
	return result.Persistent()
}

var procSelectKeys = func(args []Object) Object {
	CheckArity(args, 2, 2)
	m := args[0]
	keys := newSeqCursor(EnsureArgIsSeqable(args, 1).Seq())
	result := EmptyArrayMap().AsTransient().(TransientMapCollection)
	var associative Associative
	for keys.hasNext() {
		if associative == nil {
			associative = EnsureObjectIsAssociative(m, "")
		}
		if entry := associative.EntryAt(keys.first()); entry != nil {
			// Retain EntryAt's lookup and key-representation rules, which
			// differ between ArrayMap and HashMap.
			result = result.AssocBang(entry.At(0), entry.At(1)).(TransientMapCollection)
		}
		keys.advance()
	}
	persistent := result.Persistent()
	if metadata, ok := m.(Meta); ok {
		if meta := metadata.GetMeta(); meta != nil {
			return persistent.(Meta).WithMeta(meta)
		}
	}
	return persistent
}

var procGetIn = func(args []Object) Object {
	CheckArity(args, 2, 3)
	value := args[0]
	if len(args) == 2 {
		switch args[1].(type) {
		case *ArrayVector, *Vector:
			// These built-in reductions have ordinary indexed traversal.
		default:
			if reducer, ok := args[1].(Reduce); ok {
				return reducer.reduceInit(nativeGetReducer, value)
			}
		}
	}
	keys := newSeqCursor(EnsureArgIsSeqable(args, 1).Seq())
	for keys.hasNext() {
		key := keys.first()
		found := false
		var next Object
		if gettable, ok := value.(Gettable); ok {
			found, next = gettable.Get(key)
		}
		if !found {
			if len(args) == 3 {
				return args[2]
			}
			next = NIL
		}
		value = next
		keys.advance()
	}
	return value
}

var nativeGetReducer = Proc{Fn: func(pair []Object) Object {
	return nativeGet(pair[0], pair[1])
}}

func nativeGet(value, key Object) Object {
	if gettable, ok := value.(Gettable); ok {
		if found, result := gettable.Get(key); found {
			return result
		}
	}
	return NIL
}

type associativePathFrame struct {
	value Object
	key   Object
}

// Destructuring [k & ks] realizes the next key before looking up k. Empty
// paths use a nil key. Record parents without validating their associativity:
// assoc performs that validation on the way back out, after update's callback.
func associativePath(value Object, path Object, frames []associativePathFrame) []associativePathFrame {
	// Size known persistent inputs without invoking an arbitrary Counted
	// implementation (which may run user code or have observable effects).
	var size int
	switch keys := path.(type) {
	case *ArrayVector:
		size = keys.Count()
	case *Vector:
		size = keys.Count()
	case *List:
		size = keys.Count()
	case *ArraySeq:
		size = keys.Count()
	}
	if size > cap(frames) {
		frames = make([]associativePathFrame, 0, size)
	}
	keys := newSeqCursor(EnsureObjectIsSeqable(path, "").Seq())
	for {
		key := Object(NIL)
		present := keys.hasNext()
		if present {
			key = keys.first()
			keys.advance()
			present = keys.hasNext()
		}
		frames = append(frames, associativePathFrame{value: value, key: key})
		if !present {
			return frames
		}
		value = nativeGet(value, key)
	}
}

func rebuildAssociativePath(frames []associativePathFrame, value Object) Object {
	for i := len(frames) - 1; i >= 0; i-- {
		frame := frames[i]
		value = EnsureObjectIsAssociative(frame.value, "").Assoc(frame.key, value)
	}
	return value
}

var procAssocIn = func(args []Object) Object {
	CheckArity(args, 3, 3)
	var storage [64]associativePathFrame
	frames := associativePath(args[0], args[1], storage[:0])
	return rebuildAssociativePath(frames, args[2])
}

var procUpdateIn = func(args []Object) Object {
	CheckArity(args, 3, math.MaxInt)
	var storage [64]associativePathFrame
	frames := associativePath(args[0], args[1], storage[:0])
	leaf := frames[len(frames)-1]
	old := nativeGet(leaf.value, leaf.key)
	callbackArgs := make([]Object, len(args)-2)
	callbackArgs[0] = old
	copy(callbackArgs[1:], args[3:])
	value := EnsureArgIsCallable(args, 2).Call(callbackArgs)
	return rebuildAssociativePath(frames, value)
}
