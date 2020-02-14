package gen_go

import (
	"fmt"
	"reflect"
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
	KeySortFn      func(keys []reflect.Value)
	FieldSortFn    func(members []string)
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
	if g.KeySortFn != nil {
		g.KeySortFn(keys)
	}
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
	if g.FieldSortFn != nil {
		g.FieldSortFn(members)
	}
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
