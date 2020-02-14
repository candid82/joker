package gen_go

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
)

type GoGen struct {
	Statics        *[]string
	Runtime        *[]string
	Generated      map[interface{}]interface{} // key{reflect.Value} => map{string} that is the generated name of the var; else key{name} => map{obj}
	StructHookFn   func(target string, t reflect.Type, obj interface{}) (res string, deferredFunc func(target string, obj interface{}) string)
	TypeToStringFn func(string) string                                         // Convert stringize reflect.Type to how generate code will refer to it
	ValueHookFn    func(target string, t reflect.Type, v reflect.Value) string // return non-empty string to short-circuit value expansion
	PointerHookFn  func(target string, v reflect.Value) string                 // return non-empty string to short-circuit value expansion
}

// Generate Go code to initialize a variable (either statically or at run time) to the value specified by obj.
func (g *GoGen) Var(name string, atRuntime bool, obj interface{}) {
	if _, found := g.Generated[name]; found {
		return // Already generated.
	}

	g.Generated[name] = nil // Generation is in-progress.
	v := reflect.ValueOf(obj)

	if atRuntime {
		*g.Statics = append(*g.Statics, fmt.Sprintf(`
var %s %s`[1:],
			name, g.valueTypeToStringFn(v)))
		*g.Runtime = append(*g.Runtime, fmt.Sprintf(`
	%s = %s`[1:],
			name, g.value(name, reflect.TypeOf(nil), v)))
	} else {
		*g.Statics = append(*g.Statics, fmt.Sprintf(`
var %s %s = %s`[1:],
			name, g.valueTypeToStringFn(v), g.value(name, reflect.TypeOf(nil), v)))
	}

	g.Generated[name] = obj // Generation is complete.
}

// Generate code specifying the value as it would be assigned to a given target with a given declared type.
func (g *GoGen) value(target string, t reflect.Type, v reflect.Value) string {
	v = UnsafeReflectValue(v)
	if v.IsZero() && t == v.Type() {
		// Empty value and the target (destination) is of the same concrete type, so no need to emit anything.
		return ""
	}

	if res := g.ValueHookFn(target, t, v); res != "" {
		return res
	}

	switch v.Kind() {
	case reflect.Interface:
		return g.value(target, t, v.Elem())

	case reflect.Ptr:
		return g.pointer(target, v)

	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return fmt.Sprintf("%d", v.Int())

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return fmt.Sprintf("%d", v.Uint())

	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'x', -1, 64)

	case reflect.String:
		return strconv.Quote(v.String())

	case reflect.Slice, reflect.Array:
		return fmt.Sprintf(`%s{%s}`, g.valueTypeToStringFn(v), g.slice(target, v))

	case reflect.Map:
		typeName := g.valueTypeToStringFn(v)
		obj := v.Interface()
		if obj == nil {
			return ""
		}
		return fmt.Sprintf(`
%s{%s}`[1:],
			typeName, joinMembers(g.keysAndValues(target, typeName, obj)))

	case reflect.Struct:
		typeName := g.valueTypeToStringFn(v)
		obj := v.Interface()
		lazy := ""
		if g.StructHookFn != nil {
			res, deferredFunc := g.StructHookFn(target, t, obj)
			if res != "-" {
				return res
			}
			if deferredFunc != nil {
				defer deferredFunc(target, obj)
			}
		}
		if obj == nil {
			return ""
		}
		members := g.fields(target, typeName, obj)
		if lazy != "" {
			members = append(members, lazy)
		}
		return fmt.Sprintf(`
%s{%s}`[1:],
			typeName, joinMembers(members))

	default:
		return fmt.Sprintf("nil /* UNKNOWN TYPE obj=%T v=%s v.Kind()=%s vt=%s */", v.Interface(), v, v.Kind(), v.Type())
	}
}

// Generate key/value assignments for a map.
func (g *GoGen) keysAndValues(target string, name string, obj interface{}) (members []string) {
	v := reflect.ValueOf(obj)
	if v.IsNil() {
		return
	}
	keys := v.MapKeys()
	valueType := v.Type().Elem()
	sortValues(keys) // TODO
	for _, key := range keys {
		k := g.value("", reflect.TypeOf(nil), key)
		vi := v.MapIndex(key)
		v := g.value(fmt.Sprintf("%s[%s]%s", target, k, assertValueType(target, k, valueType, vi)), valueType, vi)
		if isNil(v) {
			continue
		}
		members = append(members, fmt.Sprintf(`
	%s: %s,`[1:],
			k, v))
	}
	return members
}

// Generate key/value assignments for fields of a structure.
func (g *GoGen) fields(target string, name string, obj interface{}) (members []string) {
	v := reflect.ValueOf(obj)
	vt := v.Type()
	numMembers := v.NumField()
	for i := 0; i < numMembers; i++ {
		vtf := vt.Field(i)
		vf := v.Field(i)
		val := g.value(fmt.Sprintf("%s.%s%s", target, vtf.Name, assertValueType(target, vtf.Name, vtf.Type, vf)), vtf.Type, vf)
		if val == "" {
			continue
		}
		members = append(members, fmt.Sprintf(`
	%s: %s,`[1:],
			vtf.Name, val))
	}
	sort.Strings(members) // TODO
	return members
}

func (g *GoGen) slice(target string, v reflect.Value) string {
	numEntries := v.Len()
	elemType := v.Type().Elem()
	el := []string{}
	for i := 0; i < numEntries; i++ {
		res := g.value(fmt.Sprintf("%s[%d]", target, i), elemType, v.Index(i))
		if res == "" {
			el = append(el, "\tnil,")
		} else {
			el = append(el, "\t"+res+",")
		}
	}
	return joinMembers(el)
}

// Generate initial value for a pointer.
func (g *GoGen) pointer(target string, ptr reflect.Value) string {
	if ptr.IsNil() {
		return "nil"
	}

	v := ptr.Elem()
	v = UnsafeReflectValue(v)

	if res := g.PointerHookFn(target, ptr); res != "" {
		return res
	}

	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return "nil"
		}
		return g.pointer(target, v.Elem())

	default:
		thing, found := g.Generated[v]
		if !found {
			ptrToObj := ptr.Interface()
			obj := v.Interface()
			name := uniqueId(ptrToObj)
			if g.LateInit { // TODO:
				if destVar, yes := ptrToObj.(*Var); yes {
					if e, isVarRefExpr := destVar.Expr().(*VarRefExpr); isVarRefExpr {
						sourceVarName := e.Var().Name()
						if _, found := knownLateInits[sourceVarName]; found {
							destVarId := uniqueId(destVar)
							*g.Runtime = append(*g.Runtime, fmt.Sprintf(`
	%s.Value = %s.Value`[1:],
								destVarId, uniqueId(e.Var())))
						}
					}
				}
			}
			g.Generated[v] = name

			if ns, yes := ptrToObj.(*Namespace); yes { // TODO:
				if _, found := namespaces[ns.ToString(false)]; found {
					oldNamespace := g.Namespace
					oldRuntime := g.Runtime
					oldImports := g.Import
					oldRequired := g.Required
					defer func() {
						g.Namespace = oldNamespace
						g.Runtime = oldRuntime
						g.Import = oldImports
						g.Required = oldRequired
					}()

					g.Namespace = ns

					rt, found := g.Runtimes[ns]
					if !found {
						newRuntime := []string{}
						rt = &newRuntime
						g.Runtimes[ns] = rt
					}
					g.Runtime = rt

					imp, found := g.Imports[ns]
					if !found {
						newImport := Imports{}
						imp = &newImport
						g.Imports[ns] = imp
					}
					g.Import = imp

					rq, found := g.Requireds[ns]
					if !found {
						newRequired := map[*Namespace]struct{}{}
						rq = &newRequired
						g.Requireds[ns] = rq
					}
					g.Required = rq
				}
			}

			g.Var(name, false, obj)
			return "&" + name
		}
		name := thing.(string)
		status, found := g.Generated[name]
		if !found {
			panic(fmt.Sprintf("cannot find generated thing %s: %+v", name, v.Interface()))
		}
		if status == nil {
			*g.Runtime = append(*g.Runtime, fmt.Sprintf(`
	%s = &%s`[1:],
				asTarget(target), name))
			return fmt.Sprintf("nil /* %s: &%s */", g.Namespace.ToString(false), name)
		}
		return "&" + name
	}
}

func assertValueType(target, name string, valueType reflect.Type, r reflect.Value) string {
	if r.IsZero() {
		return ""
	}
	if r.Kind() == reflect.Interface {
		r = r.Elem()
	}
	if valueType == r.Type() {
		return ""
	}
	return ".(" + g.valueTypeToStringFn(r) + ")"
}

func (g *GoGen) valueTypeToStringFn(v reflect.Value) string {
	return g.TypeToStringFn(v.Type().String())
}

func joinMembers(members []string) string {
	f := strings.Join(members, "\n")
	if f != "" {
		f = "\n" + f + "\n"
	}
	return f
}

func isNil(s string) bool {
	return s == "" || s == "nil" || strings.HasPrefix(s, "nil /*")
}

// NOTE: Below this line, code comes from github.com/jcburley/go-spew:

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
const flagPrivate = 0x20

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
var flagValOffset = func() uintptr {
	field, ok := reflect.TypeOf(reflect.Value{}).FieldByName("flag")
	if !ok {
		panic("reflect.Value has no flag field")
	}
	return field.Offset
}()

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
type flag uintptr

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
func flagField(v *reflect.Value) *flag {
	return (*flag)(unsafe.Pointer(uintptr(unsafe.Pointer(v)) + flagValOffset))
}

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
func UnsafeReflectValue(v reflect.Value) reflect.Value {
	if !v.IsValid() || (v.CanInterface() && v.CanAddr()) {
		return v
	}
	flagFieldPtr := flagField(&v)
	*flagFieldPtr &^= flagPrivate
	return v
}

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
	values  []reflect.Value
	strings []string // either nil or same len and values
}

// newValuesSorter initializes a valuesSorter instance, which holds a set of
// surrogate keys on which the data should be sorted.  It uses flags in
// ConfigState to decide if and how to populate those surrogate keys.
func newValuesSorter(values []reflect.Value) sort.Interface {
	vs := &valuesSorter{values: values}
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
func valueSortLess(a, b reflect.Value) bool {
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
		if aIndex, found := namespaceIndices[aStr]; found {
			aStr = fmt.Sprintf("[%03d]%s", aIndex, aStr)
		}
		if bIndex, found := namespaceIndices[bStr]; found {
			bStr = fmt.Sprintf("[%03d]%s", bIndex, bStr)
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
			return valueSortLess(av, bv)
		}
	case reflect.Ptr:
		return valueSortLess(a.Elem(), b.Elem())
	}
	return a.String() < b.String()
}

// Less returns whether the value at index i should sort before the
// value at index j.  It is part of the sort.Interface implementation.
func (s *valuesSorter) Less(i, j int) bool {
	if s.strings == nil {
		return valueSortLess(s.values[i], s.values[j])
	}
	return s.strings[i] < s.strings[j]
}

// sortValues is a sort function that handles both native types and any type that
// can be converted to error or Stringer.  Other inputs are sorted according to
// their Value.String() value to ensure display stability.
func sortValues(values []reflect.Value) {
	if len(values) == 0 {
		return
	}
	sort.Sort(newValuesSorter(values))
}
