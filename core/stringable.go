package core

func EnsureObjectIsStringable(obj Object, pattern string) String {
	switch c := obj.(type) {
	case String:
		return c
	case Char:
		return String{S: string(c.Ch)}
	default:
		panic(FailObject(c, "Stringable", pattern))
	}
}

func EnsureArgIsStringable(args []Object, index int) String {
	switch c := args[index].(type) {
	case String:
		return c
	case Char:
		return String{S: string(c.Ch)}
	default:
		panic(FailArg(c, "Stringable", index))
	}
}
