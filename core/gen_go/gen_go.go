package gen_go

// Implements (*GenGo) Var(), which generates Go code that (mostly)
// statically initializes the contents of the provided argument.
//
// This is hardly suitable as a general-purpose static-code
// generator. Besides simple things like (say) a uint8 of 1 being
// initialized as just its value (1) instead of uint8(1), there are
// various data types, potentially crucial hooks, and other behaviors
// that are not implemented simply because Joker doesn't need them.

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type GenGo struct {
	Statics        *[]string
	Runtime        *[]string
	Generated      map[interface{}]interface{} // key{reflect.Value} => map{string} that is the generated name of the var; else key{name} => map{obj}
	TypeToStringFn func(string) string         // Convert stringize reflect.Type to how generate code will refer to it
	KeySortFn      func(keys []reflect.Value)
	FieldSortFn    func(members []string)
	WhereFn        func() string
	StructHookFn   func(target string, t reflect.Type, obj interface{}) (res string, deferredFunc func(target string, obj interface{}))
	ValueHookFn    func(target string, t reflect.Type, v reflect.Value) string // return non-empty string to short-circuit value expansion
	PointerHookFn  func(target string, ptr, v reflect.Value) string            // return non-empty string to short-circuit value expansion
	PtrToValueFn   func(ptr, v reflect.Value) string                           // typically calls (*GenGo)Var() to generate the variable and returns that result
}

// Generate Go code to initialize a variable (either statically or at run time) to the value specified by obj.
func (g *GenGo) Var(name string, atRuntime bool, obj interface{}) {
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
func (g *GenGo) value(target string, t reflect.Type, v reflect.Value) string {
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
func (g *GenGo) keysAndValues(target string, name string, obj interface{}) (members []string) {
	v := reflect.ValueOf(obj)
	if v.IsNil() {
		return
	}
	keys := v.MapKeys()
	valueType := v.Type().Elem()
	if g.KeySortFn != nil {
		g.KeySortFn(keys)
	}
	for _, key := range keys {
		k := g.value("", reflect.TypeOf(nil), key)
		vi := v.MapIndex(key)
		v := g.value(fmt.Sprintf("%s[%s]%s", target, k, g.assertValueType(target, k, valueType, vi)), valueType, vi)
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
func (g *GenGo) fields(target string, name string, obj interface{}) (members []string) {
	v := reflect.ValueOf(obj)
	vt := v.Type()
	numMembers := v.NumField()
	for i := 0; i < numMembers; i++ {
		vtf := vt.Field(i)
		vf := v.Field(i)
		val := g.value(fmt.Sprintf("%s.%s%s", target, vtf.Name, g.assertValueType(target, vtf.Name, vtf.Type, vf)), vtf.Type, vf)
		if val == "" {
			continue
		}
		members = append(members, fmt.Sprintf(`
	%s: %s,`[1:],
			vtf.Name, val))
	}
	if g.FieldSortFn != nil {
		g.FieldSortFn(members)
	}
	return members
}

func (g *GenGo) slice(target string, v reflect.Value) string {
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
func (g *GenGo) pointer(target string, ptr reflect.Value) string {
	if ptr.IsNil() {
		return "nil"
	}

	v := ptr.Elem()
	v = UnsafeReflectValue(v)

	if res := g.PointerHookFn(target, ptr, v); res != "" {
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
			return g.PtrToValueFn(ptr, v)
		}
		name := thing.(string)
		status, found := g.Generated[name]
		if !found {
			panic(fmt.Sprintf("cannot find generated thing %s: %+v", name, v.Interface()))
		}
		if status == nil {
			// Compilation in progress, so assign this at runtime to avoid cycles that Go doesn't like.
			*g.Runtime = append(*g.Runtime, fmt.Sprintf(`
	%s = &%s`[1:],
				AsTarget(target), name))
			return fmt.Sprintf("nil /* %s: &%s */", g.WhereFn(), name)
		}
		return "&" + name
	}
}

func (g *GenGo) assertValueType(target, name string, valueType reflect.Type, r reflect.Value) string {
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

func (g *GenGo) valueTypeToStringFn(v reflect.Value) string {
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

func AsTarget(s string) string {
	if s == "" || s[len(s)-1] != ')' {
		return s
	}
	ix := strings.LastIndex(s, ".(")
	if ix < 0 {
		return s
	}
	return s[0:ix]
}
