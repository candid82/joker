package core

// walkChildren implements joker.walk's reconstruction rules. In particular,
// map entries are ordinary vectors, nil is a leaf despite implementing Seq in
// Go, and non-list seqs remain fully realized lazy sequences (not lists).
func walkChildren(form Object, inner func(Object) Object) Object {
	switch value := form.(type) {
	case Nil:
		return form
	case *List:
		items := make([]Object, 0, value.Count())
		for cursor := newSeqCursor(value); cursor.hasNext(); cursor.advance() {
			items = append(items, inner(cursor.first()))
		}
		return NewListFrom(items...)
	case Seq:
		mapped := NewMapSeq(Proc{Fn: func(args []Object) Object {
			return inner(args[0])
		}}, value)
		// Match (doall (map inner form)), including the realized empty tail.
		// Realization clears each node's callback, so the returned value owns
		// neither a traversal closure nor an execution context.
		for seq := mapped; !seq.IsEmpty(); seq = seq.Rest() {
		}
		return mapped
	case Collection:
		seed := value.Empty()
		cursor := newSeqCursor(value.Seq())
		if editable, ok := seed.(Editable); ok {
			builder := editable.AsTransient()
			for cursor.hasNext() {
				child := inner(cursor.first())
				cursor.advance()
				builder = builder.ConjBang(child)
			}
			// Match into, which restores the metadata of (empty form), not
			// the original form. Never expose the editable accumulator.
			return builder.Persistent().(Meta).WithMeta(seed.(Meta).GetMeta())
		}
		var result Object = seed
		for cursor.hasNext() {
			child := inner(cursor.first())
			cursor.advance()
			result = procConj([]Object{result, child})
		}
		return result
	default:
		return form
	}
}

// Resolve callbacks on invocation. Callable.Call synchronously re-enters the
// ambient VM, including lazy-source realization and nested native calls.
// Traversal never releases the GIL or
// captures the caller's execution on a returned value.
func walkCallback(callback Object) func(Object) Object {
	var callable Callable
	return func(form Object) Object {
		if callable == nil {
			callable = EnsureObjectIsCallable(callback, "")
		}
		return callable.Call([]Object{form})
	}
}

var procWalk = func(args []Object) Object {
	CheckArity(args, 3, 3)
	inner := walkCallback(args[0])
	switch args[2].(type) {
	case Nil:
		// Nil is a leaf, so walk never evaluates (map inner form).
	case Seq, Collection:
		// map checks its callback even for an empty source. Scalars, on
		// the other hand, do not use or validate inner at all.
		callable := EnsureArgIsCallable(args, 0)
		inner = func(form Object) Object { return callable.Call([]Object{form}) }
	}
	result := walkChildren(args[2], inner)
	return walkCallback(args[1])(result)
}

var procPostwalk = func(args []Object) Object {
	CheckArity(args, 2, 2)
	callback := walkCallback(args[0])
	var visit func(Object) Object
	visit = func(form Object) Object {
		return callback(walkChildren(form, visit))
	}
	return visit(args[1])
}

var procPrewalk = func(args []Object) Object {
	CheckArity(args, 2, 2)
	callback := walkCallback(args[0])
	var visit func(Object) Object
	visit = func(form Object) Object {
		return walkChildren(callback(form), visit)
	}
	return visit(args[1])
}
