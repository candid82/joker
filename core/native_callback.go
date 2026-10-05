package core

// Fn and built-in collection callables consume their argument slices during
// Call. Reuse scratch storage for them, but give arbitrary host callbacks an
// owned slice: they may retain it, or re-enter this same lazy transformation.
type nativeCallback struct {
	value    Object
	callable Callable
	args     []Object
}

func newNativeCallback(value Object, arity int) *nativeCallback {
	return &nativeCallback{value: value, args: make([]Object, arity)}
}

func (callback *nativeCallback) call() Object {
	if callback.callable == nil {
		callback.callable = EnsureObjectIsCallable(callback.value, "")
	}
	return callNativeCallback(callback.callable, callback.args)
}

func callNativeCallback(callback Callable, args []Object) Object {
	switch f := callback.(type) {
	case *Var:
		return callNativeCallback(EnsureObjectIsCallable(f.Resolve(), ""), args)
	case *Fn:
		return f.Call(args)
	case Keyword, Symbol, *ArrayMap, *HashMap, *MapSet, *ArrayVector, *Vector:
		return callback.Call(args)
	default:
		return callback.Call(append([]Object(nil), args...))
	}
}
