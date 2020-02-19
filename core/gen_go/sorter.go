package gen_go

// Implements the SortValues() function, used primarily to stabilize
// (and, in some cases, ensure correct operation of) traversal of maps
// when generating code.

// NOTE: This code comes from github.com/jcburley/go-spew (originally davecgh/go-spew):

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
)

// catchPanic handles any panics that might occur during the handleMethods
// calls.
func catchPanic(w io.Writer, v reflect.Value) {
	if err := recover(); err != nil {
		w.Write([]byte("(PANIC="))
		fmt.Fprintf(w, "%v", err)
		w.Write([]byte(")"))
	}
}

// handleMethods attempts to call the Error and String methods on the underlying
// type the passed reflect.Value represents and outputes the result to Writer w.
//
// It handles panics in any called methods by catching and displaying the error
// as the formatted value.
func handleMethods(w io.Writer, v reflect.Value) (handled bool) {
	// We need an interface to check if the type implements the error or
	// Stringer interface.  However, the reflect package won't give us an
	// interface on certain things like unexported struct fields in order
	// to enforce visibility rules.  We use unsafe, when it's available,
	// to bypass these restrictions since this package does not mutate the
	// values.
	if !v.CanInterface() {
		v = UnsafeReflectValue(v)
	}

	// Choose whether or not to do error and Stringer interface lookups against
	// the base type or a pointer to the base type depending on settings.
	// Technically calling one of these methods with a pointer receiver can
	// mutate the value, however, types which choose to satisify an error or
	// Stringer interface with a pointer receiver should not be mutating their
	// state inside these interface methods.
	if !v.CanAddr() {
		v = UnsafeReflectValue(v)
	}
	if v.CanAddr() {
		v = v.Addr()
	}

	// Is it an error or Stringer?
	switch iface := v.Interface().(type) {
	case error:
		defer catchPanic(w, v)
		w.Write([]byte(iface.Error()))
		return true

	case fmt.Stringer:
		defer catchPanic(w, v)
		w.Write([]byte(iface.String()))
		return true
	}
	return false
}

// valuesSorter implements sort.Interface to allow a slice of reflect.Value
// elements to be sorted.
type valuesSorter struct {
	values               []reflect.Value
	strings              []string            // either nil or same len and values
	stringSubstitutionFn func(string) string // Mainly for substituting "[nnn]joker.whatever" for "joker.whatever" to enforce ordering
}

// newValuesSorter initializes a valuesSorter instance, which holds a set of
// surrogate keys on which the data should be sorted.  It uses flags in
// ConfigState to decide if and how to populate those surrogate keys.
func newValuesSorter(values []reflect.Value, stringSubstitutionFn func(string) string) sort.Interface {
	vs := &valuesSorter{values: values, stringSubstitutionFn: stringSubstitutionFn}
	if canSortSimply(vs.values[0]) {
		return vs
	}
	vs.strings = make([]string, len(values))
	for i := range vs.values {
		b := bytes.Buffer{}
		if !handleMethods(&b, vs.values[i]) {
			vs.strings = nil
			break
		}
		vs.strings[i] = b.String()
	}
	if vs.strings == nil {
		vs.strings = make([]string, len(values))
		for i := range vs.values {
			v := UnsafeReflectValue(vs.values[i])
			vs.strings[i] = fmt.Sprintf("%#v", v.Interface())
		}
	}
	return vs
}

// canSortSimply tests whether a reflect.Kind is a primitive that can be sorted
// directly, or whether it should be considered for sorting by surrogate keys
// (if the ConfigState allows it).
func canSortSimply(value reflect.Value) bool {
	// This switch parallels valueSortLess, except for the default case.
	switch value.Kind() {
	case reflect.Bool:
		return true
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return true
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	case reflect.String:
		return true
	case reflect.Uintptr:
		return true
	case reflect.Array:
		return true
	case reflect.Ptr:
		return canSortSimply(value.Elem())
	}
	return false
}

// Len returns the number of values in the slice.  It is part of the
// sort.Interface implementation.
func (s *valuesSorter) Len() int {
	return len(s.values)
}

// Swap swaps the values at the passed indices.  It is part of the
// sort.Interface implementation.
func (s *valuesSorter) Swap(i, j int) {
	s.values[i], s.values[j] = s.values[j], s.values[i]
	if s.strings != nil {
		s.strings[i], s.strings[j] = s.strings[j], s.strings[i]
	}
}

// valueSortLess returns whether the first value should sort before the second
// value.  It is used by valueSorter.Less as part of the sort.Interface
// implementation.
func valueSortLess(a, b reflect.Value, stringSubstitutionFn func(string) string) bool {
	switch a.Kind() {
	case reflect.Bool:
		return !a.Bool() && b.Bool()
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return a.Int() < b.Int()
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return a.Uint() < b.Uint()
	case reflect.Float32, reflect.Float64:
		return a.Float() < b.Float()
	case reflect.String:
		aStr := a.String()
		bStr := b.String()
		if stringSubstitutionFn != nil {
			aStr = stringSubstitutionFn(aStr)
			bStr = stringSubstitutionFn(bStr)
		}
		return aStr < bStr
	case reflect.Uintptr:
		return a.Uint() < b.Uint()
	case reflect.Array:
		// Compare the contents of both arrays.
		l := a.Len()
		for i := 0; i < l; i++ {
			av := a.Index(i)
			bv := b.Index(i)
			if av.Interface() == bv.Interface() {
				continue
			}
			return valueSortLess(av, bv, stringSubstitutionFn)
		}
	case reflect.Ptr:
		// Assume both are pointers (true for Joker)
		return valueSortLess(a.Elem(), b.Elem(), stringSubstitutionFn)
	}
	return a.String() < b.String()
}

// Less returns whether the value at index i should sort before the
// value at index j.  It is part of the sort.Interface implementation.
func (s *valuesSorter) Less(i, j int) bool {
	if s.strings == nil {
		return valueSortLess(s.values[i], s.values[j], s.stringSubstitutionFn)
	}
	return s.strings[i] < s.strings[j]
}

// SortValues is a sort function that handles both native types and any type that
// can be converted to error or Stringer.  Other inputs are sorted according to
// their Value.String() value to ensure display stability.
func SortValues(values []reflect.Value, stringSubstitutionFn func(string) string) {
	if len(values) == 0 {
		return
	}
	sort.Sort(newValuesSorter(values, stringSubstitutionFn))
}
