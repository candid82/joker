package core

import (
	"sort"
	"strconv"
	"strings"
)

// ArityError describes a rejected argument count. Runtime checks use the
// function's implementation; the linter may also use declared :arglists.
type ArityError struct {
	Actual      int
	Fixed       []int
	VariadicMin int // -1 if there is no variadic arity
	name        string
}

// ExpectedString describes accepted counts, collapsing a contiguous variadic
// range to "at least N" while preserving gaps in the supported arities.
func (err *ArityError) ExpectedString() string {
	fixed := append([]int(nil), err.Fixed...)
	sort.Ints(fixed)
	min := err.VariadicMin
	if min >= 0 {
		for i := len(fixed) - 1; i >= 0; i-- {
			if fixed[i] == min-1 {
				min--
			}
		}
	}

	var expected []string
	for i, n := range fixed {
		if (min >= 0 && n >= min) || (i > 0 && n == fixed[i-1]) {
			continue
		}
		expected = append(expected, strconv.Itoa(n))
	}
	if min >= 0 {
		expected = append(expected, "at least "+strconv.Itoa(min))
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

func newFnExprArityError(expr *FnExpr, argc int) *ArityError {
	err := &ArityError{Actual: argc, VariadicMin: -1, name: expr.functionName()}
	for _, a := range expr.arities {
		err.Fixed = append(err.Fixed, len(a.args))
	}
	if expr.variadic != nil {
		err.VariadicMin = len(expr.variadic.args) - 1
	}
	return err
}

func newFnSummaryArityError(summary *FnSummary, argc int) *ArityError {
	err := &ArityError{Actual: argc, VariadicMin: -1, name: summary.functionName()}
	for _, a := range summary.arities {
		err.Fixed = append(err.Fixed, a.argCount)
	}
	if summary.variadic != nil {
		err.VariadicMin = summary.variadic.argCount - 1
	}
	return err
}

func newArglistArityError(arglist Seq, argc int, name string) *ArityError {
	err := &ArityError{Actual: argc, VariadicMin: -1, name: name}
	for !arglist.IsEmpty() {
		if v, ok := arglist.First().(Vec); ok {
			if n := v.Count(); n >= 2 && v.Nth(n-2).Equals(SYMBOLS.amp) {
				min := n - 2
				if err.VariadicMin < 0 || min < err.VariadicMin {
					err.VariadicMin = min
				}
			} else {
				err.Fixed = append(err.Fixed, v.Count())
			}
		}
		arglist = arglist.Rest()
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
