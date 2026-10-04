package core

import (
	"strconv"
	"strings"
)

// ArityError describes a rejected argument count using the function's actual
// implementation, not its optional :arglists metadata.
type ArityError struct {
	Actual      int
	Fixed       []int
	VariadicMin int // -1 if there is no variadic arity
	name        string
}

// ExpectedString lists accepted counts, using a trailing + for a variadic minimum.
func (err *ArityError) ExpectedString() string {
	var expected []string
	for _, n := range err.Fixed {
		expected = append(expected, strconv.Itoa(n))
	}
	if err.VariadicMin >= 0 {
		expected = append(expected, strconv.Itoa(err.VariadicMin)+"+")
	}
	return strings.Join(expected, " or ")
}

func (err *ArityError) Error() string {
	return "Wrong number of args (" + strconv.Itoa(err.Actual) + ") passed to " + err.name + ", expected: " + err.ExpectedString()
}

func newArityError(proto *FunctionProto, argc int) *ArityError {
	err := &ArityError{Actual: argc, VariadicMin: -1, name: proto.Name}
	for _, a := range proto.Arities {
		err.Fixed = append(err.Fixed, a.Arity)
	}
	if a := proto.VariadicArity; a != nil {
		err.VariadicMin = a.Arity
	} else if len(proto.Arities) == 0 && proto.Chunk != nil {
		err.Fixed = []int{0}
	}
	return err
}

// CheckArity returns nil when argc is accepted. Like Call, it compiles parsed
// functions on demand and may panic if compilation fails. The caller must hold
// the GIL.
func (fn *Fn) CheckArity(argc int) *ArityError {
	fn.ensureCompiled()
	if selectArityProto(fn.proto, argc) != nil {
		return nil
	}
	return newArityError(fn.proto, argc)
}
