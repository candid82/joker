package gen_go

import (
	"reflect"
)

type GoGen struct {
	Statics   *[]string
	Runtime   *[]string
	Generated map[interface{}]interface{} // key{reflect.Value} => map{string} that is the generated name of the var; else key{name} => map{obj}
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
			name, coreTypeName(v)))
		*g.Runtime = append(*g.Runtime, fmt.Sprintf(`
	%s = %s`[1:],
			name, g.value(name, reflect.TypeOf(nil), v)))
	} else {
		*g.Statics = append(*g.Statics, fmt.Sprintf(`
var %s %s = %s`[1:],
			name, coreTypeName(v), g.value(name, reflect.TypeOf(nil), v)))
	}

	g.Generated[name] = obj // Generation is complete.
}

// Generate
func (g *GoGen) value(target string, t reflect.Type, v reflect.Value) string {
	v = UnsafeReflectValue(v)
	if v.IsZero() && t == v.Type() {
		// Empty value and the target (destination) is of the same concrete type, so no need to emit anything.
		return ""
	}

	switch pkg := path.Base(v.Type().PkgPath()); pkg {
	case "reflect":
		t := coreTypeString(fmt.Sprintf("%s", v))
		components := strings.Split(t, ".")
		if len(components) == 2 {
			// not handling more than one component yet!
			importedAs := AddImport(g.StaticImport, "", components[0], true)
			t = fmt.Sprintf("%s.%s", importedAs, components[1])
		}
		el := ""
		if t[0] != '*' {
			t = "*" + t
			el = ".Elem()"
		}
		importedAs := AddImport(g.StaticImport, "", "reflect", true)
		return fmt.Sprintf("%s.TypeOf((%s)(nil))%s", importedAs, t, el)
	case "core":
	case ".":
	default:
		panic(fmt.Sprintf("unexpected PkgPath `%s' for %+v", pkg, v.Interface()))
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
		return fmt.Sprintf(`%s{%s}`, coreTypeName(v), g.slice(target, v))

	case reflect.Map:
		typeName := coreTypeName(v)
		obj := v.Interface()
		if obj == nil {
			return ""
		}
		return fmt.Sprintf(`
%s{%s}`[1:],
			typeName, joinMembers(g.keysAndValues(target, typeName, obj)))

	case reflect.Struct:
		typeName := coreTypeName(v)
		obj := v.Interface()
		lazy := ""
		switch obj := obj.(type) { // TODO:
		case Proc:
			return g.emitProc(target, obj) // TODO
		case Namespace:
			nsName := obj.Name.Name()
			if VerbosityLevel > 0 {
				fmt.Printf("COMPILING %s\n", nsName)
			}
			lateInit := g.LateInit
			g.LateInit = nsName != "joker.core"
			defer func() {
				g.LateInit = lateInit
				if VerbosityLevel > 0 {
					fmt.Printf("FINISHED %s\n", nsName)
				}
			}()
		case VarRefExpr:
			if curRequired := g.Required; curRequired != nil {
				if vr := obj.Var(); vr != nil {
					if ns := vr.Namespace(); ns != nil && ns != g.Namespace && ns != GLOBAL_ENV.CoreNamespace {
						(*curRequired)[ns] = struct{}{}
					}
				}
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
}

// Generate key/value assignments for fields of a structure.
func (g *GoGen) fields(target string, name string, obj interface{}) (members []string) {
	v := reflect.ValueOf(obj)
	vt := v.Type()
	numMembers := v.NumField()
	for i := 0; i < numMembers; i++ {
		vtf := vt.Field(i)
		vf := v.Field(i)
		val := g.emitValue(fmt.Sprintf("%s.%s%s", target, vtf.Name, assertValueType(target, vtf.Name, vtf.Type, vf)), vtf.Type, vf)
		if val == "" {
			continue
		}
		members = append(members, fmt.Sprintf(`
	%s: %s,`[1:],
			vtf.Name, val))
	}
	sort.Strings(members) // TODO
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

	// TODO:
	switch pkg := path.Base(v.Type().PkgPath()); pkg {
	case "regexp":
		return g.emitPtrToRegexp(target, ptr)
	case "core":
	case ".":
	default:
		panic(fmt.Sprintf("unexpected PkgPath `%s' for &%+v", pkg, v.Interface()))
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

			g.emitVar(name, false, obj)
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
