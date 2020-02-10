package gen_go

import (
	"reflect"
)

type GoGen struct {
	Statics      *[]string
	StaticImport *Imports
	Runtime      *[]string
	Import       *Imports
	Generated    map[interface{}]interface{} // key{reflect.Value} => map{string} that is the generated name of the var; else key{name} => map{obj}
}

func (goGen *GoGen) Var(name string, atRuntime bool, obj interface{}) {
	if _, found := goGen.Generated[name]; found {
		return // Already generated.
	}
	goGen.Generated[name] = nil
	v := reflect.ValueOf(obj)

	if atRuntime {
		*goGen.Statics = append(*goGen.Statics, fmt.Sprintf(`
var %s %s`[1:],
			name, coreTypeName(v)))
		*goGen.Runtime = append(*goGen.Runtime, fmt.Sprintf(`
	%s = %s`[1:],
			name, goGen.emitValue(name, reflect.TypeOf(nil), v)))
	} else {
		*goGen.Statics = append(*goGen.Statics, fmt.Sprintf(`
var %s %s = %s`[1:],
			name, coreTypeName(v), goGen.emitValue(name, reflect.TypeOf(nil), v)))
	}
	goGen.Generated[name] = obj
}

func (goGen *GoGen) emitMembers(target string, name string, obj interface{}) (members []string) {
	v := reflect.ValueOf(obj)
	kind := v.Kind()
	switch kind {
	case reflect.Map:
		if v.IsNil() {
			return
		}
		keys := v.MapKeys()
		valueType := v.Type().Elem()
		sortValues(keys)
		for _, key := range keys {
			k := goGen.emitValue("", reflect.TypeOf(nil), key)
			vi := v.MapIndex(key)
			v := goGen.emitValue(fmt.Sprintf("%s[%s]%s", target, k, assertValueType(target, k, valueType, vi)), valueType, vi)
			if isNil(v) {
				continue
			}
			members = append(members, fmt.Sprintf(`
	%s: %s,`[1:],
				k, v))
		}
	case reflect.Struct:
		vt := v.Type()
		numMembers := v.NumField()
		for i := 0; i < numMembers; i++ {
			vtf := vt.Field(i)
			vf := v.Field(i)
			val := goGen.emitValue(fmt.Sprintf("%s.%s%s", target, vtf.Name, assertValueType(target, vtf.Name, vtf.Type, vf)), vtf.Type, vf)
			if val == "" {
				continue
			}
			members = append(members, fmt.Sprintf(`
	%s: %s,`[1:],
				vtf.Name, val))
		}
		sort.Strings(members)
	default:
		panic(fmt.Sprintf("unsupported type %T for %s", obj, name))
	}
	return
}

func (goGen *GoGen) emitValue(target string, t reflect.Type, v reflect.Value) string {
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
			importedAs := AddImport(goGen.StaticImport, "", components[0], true)
			t = fmt.Sprintf("%s.%s", importedAs, components[1])
		}
		el := ""
		if t[0] != '*' {
			t = "*" + t
			el = ".Elem()"
		}
		importedAs := AddImport(goGen.StaticImport, "", "reflect", true)
		return fmt.Sprintf("%s.TypeOf((%s)(nil))%s", importedAs, t, el)
	case "core":
	case ".":
	default:
		panic(fmt.Sprintf("unexpected PkgPath `%s' for %+v", pkg, v.Interface()))
	}

	switch v.Kind() {
	case reflect.Interface:
		return goGen.emitValue(target, t, v.Elem())

	case reflect.Ptr:
		return goGen.emitPtrTo(target, v)

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
		return fmt.Sprintf(`%s{%s}`, coreTypeName(v), goGen.emitSlice(target, v))

	case reflect.Struct:
		typeName := coreTypeName(v)
		obj := v.Interface()
		lazy := ""
		switch obj := obj.(type) {
		case Proc:
			return goGen.emitProc(target, obj)
		case Namespace:
			nsName := obj.Name.Name()
			if VerbosityLevel > 0 {
				fmt.Printf("COMPILING %s\n", nsName)
			}
			lateInit := goGen.LateInit
			goGen.LateInit = nsName != "joker.core"
			defer func() {
				goGen.LateInit = lateInit
				if VerbosityLevel > 0 {
					fmt.Printf("FINISHED %s\n", nsName)
				}
			}()
		case VarRefExpr:
			if curRequired := goGen.Required; curRequired != nil {
				if vr := obj.Var(); vr != nil {
					if ns := vr.Namespace(); ns != nil && ns != goGen.Namespace && ns != GLOBAL_ENV.CoreNamespace {
						(*curRequired)[ns] = struct{}{}
					}
				}
			}
		}
		if obj == nil {
			return ""
		}
		members := goGen.emitMembers(target, typeName, obj)
		if lazy != "" {
			members = append(members, lazy)
		}
		return fmt.Sprintf(`
%s{%s}`[1:],
			typeName, joinMembers(members))

	case reflect.Map:
		typeName := coreTypeName(v)
		obj := v.Interface()
		if obj == nil {
			return ""
		}
		return fmt.Sprintf(`
%s{%s}`[1:],
			typeName, joinMembers(goGen.emitMembers(target, typeName, obj)))

	default:
		return fmt.Sprintf("nil /* UNKNOWN TYPE obj=%T v=%s v.Kind()=%s vt=%s */", v.Interface(), v, v.Kind(), v.Type())
	}
}

func (goGen *GoGen) emitPtrTo(target string, ptr reflect.Value) string {
	if ptr.IsNil() {
		return "nil"
	}

	v := ptr.Elem()
	v = UnsafeReflectValue(v)

	switch pkg := path.Base(v.Type().PkgPath()); pkg {
	case "regexp":
		return goGen.emitPtrToRegexp(target, ptr)
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
		return goGen.emitPtrTo(target, v.Elem())

	default:
		thing, found := goGen.Generated[v]
		if !found {
			ptrToObj := ptr.Interface()
			obj := v.Interface()
			name := uniqueId(ptrToObj)
			if goGen.LateInit {
				if destVar, yes := ptrToObj.(*Var); yes {
					if e, isVarRefExpr := destVar.Expr().(*VarRefExpr); isVarRefExpr {
						sourceVarName := e.Var().Name()
						if _, found := knownLateInits[sourceVarName]; found {
							destVarId := uniqueId(destVar)
							*goGen.Runtime = append(*goGen.Runtime, fmt.Sprintf(`
	%s.Value = %s.Value`[1:],
								destVarId, uniqueId(e.Var())))
						}
					}
				}
			}
			goGen.Generated[v] = name

			if ns, yes := ptrToObj.(*Namespace); yes {
				if _, found := namespaces[ns.ToString(false)]; found {
					oldNamespace := goGen.Namespace
					oldRuntime := goGen.Runtime
					oldImports := goGen.Import
					oldRequired := goGen.Required
					defer func() {
						goGen.Namespace = oldNamespace
						goGen.Runtime = oldRuntime
						goGen.Import = oldImports
						goGen.Required = oldRequired
					}()

					goGen.Namespace = ns

					rt, found := goGen.Runtimes[ns]
					if !found {
						newRuntime := []string{}
						rt = &newRuntime
						goGen.Runtimes[ns] = rt
					}
					goGen.Runtime = rt

					imp, found := goGen.Imports[ns]
					if !found {
						newImport := Imports{}
						imp = &newImport
						goGen.Imports[ns] = imp
					}
					goGen.Import = imp

					rq, found := goGen.Requireds[ns]
					if !found {
						newRequired := map[*Namespace]struct{}{}
						rq = &newRequired
						goGen.Requireds[ns] = rq
					}
					goGen.Required = rq
				}
			}

			goGen.emitVar(name, false, obj)
			return "&" + name
		}
		name := thing.(string)
		status, found := goGen.Generated[name]
		if !found {
			panic(fmt.Sprintf("cannot find generated thing %s: %+v", name, v.Interface()))
		}
		if status == nil {
			*goGen.Runtime = append(*goGen.Runtime, fmt.Sprintf(`
	%s = &%s`[1:],
				asTarget(target), name))
			return fmt.Sprintf("nil /* %s: &%s */", goGen.Namespace.ToString(false), name)
		}
		return "&" + name
	}
}

func (goGen *GoGen) emitSlice(target string, v reflect.Value) string {
	numEntries := v.Len()
	elemType := v.Type().Elem()
	el := []string{}
	for i := 0; i < numEntries; i++ {
		res := goGen.emitValue(fmt.Sprintf("%s[%d]", target, i), elemType, v.Index(i))
		if res == "" {
			el = append(el, "\tnil,")
		} else {
			el = append(el, "\t"+res+",")
		}
	}
	return joinMembers(el)
}
