package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	. "github.com/candid82/joker/core"
	_ "github.com/candid82/joker/std/html"
	_ "github.com/candid82/joker/std/string"
)

const hextable = "0123456789abcdef"
const masterFile = "a_code.go"
const codePattern = "a_%s_code.go"
const dataPattern = "a_%s_data.go"

type GenEnv struct {
	Statics        *[]string
	StaticImport   *Imports
	Runtime        *[]string
	Required       *map[*Namespace]struct{} // Namespaces referenced by current one
	Import         *Imports
	Namespace      *Namespace // In which the core.Var currently being emitted is said to reside
	Runtimes       map[*Namespace]*[]string
	Imports        map[*Namespace]*Imports
	Generated      map[interface{}]interface{}             // key{reflect.Value} => map{string} that is the generated name of the var; else key{name} => map{obj}
	CoreNamespaces map[string]struct{}                     // Set of the core namespaces (this excludes user and dependent std namespaces such as joker.string and joker.http)
	Requireds      map[*Namespace]*map[*Namespace]struct{} // Namespaces referenced by each namespace
	LateInit       bool                                    // Whether emitting a namespace other than joker.core
}

var (
	/* NewEnv() statically declares numerous variables and sets
	/* some of them to initial values before any func init()'s
	/* run. Later, func main() calls (*Env)InitEnv() and other
	/* receivers, as appropriate, to set per-invocation
	/* values. Normally, by this point in time, core.joke (in
	/* digested form) has already been "run", so it cannot
	/* directly use such values; but the other *.joke files won't
	/* have been run yet, so can use those values. However, that
	/* won't work here (in gen_code), since there's no
	/* corresponding main.go and (of course) the
	/* per-invocation-of-Joker values won't be known. So
	/* e.g. `(def *foo* *out*)`, while it'd work normally, will
	/* end up setting `*foo*` to nil; and (*Env)InitEnv() won't
	/* know to fix that up after setting `*out*` to a non-nil
	/* value. This map lists these "late-initialization" variables
	/* so code to set them is emitted in the respective
	/* namespace's func *Init() code. Any variable defined in
	/* terms of one of these variables is thus handled and, in
	/* turn, added to this map. Currently, only direct assignment
	/* is handled; something like `(def *n* (count
	/* *command-line-args*))` would thus not work without special
	/* handling, which does not yet appear to be necessary. */

	knownLateInits = map[string]struct{}{
		"joker.core/*in*":                struct{}{},
		"joker.core/*out*":               struct{}{},
		"joker.core/*err*":               struct{}{},
		"joker.core/*command-line-args*": struct{}{},
		"joker.core/*classpath*":         struct{}{},
		"joker.core/*core-namespaces*":   struct{}{},
		"joker.core/*verbose*":           struct{}{},
		"joker.core/*file*":              struct{}{},
		"joker.core/*main-file*":         struct{}{},
	}
)

func parseArgs(args []string) {
	length := len(args)
	stop := false
	missing := false
	var i int
	for i = 1; i < length; i++ { // shift
		switch args[i] {
		case "--verbose":
			Verbose++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(Stderr, "Error: Unrecognized option '%s'\n", args[i])
				os.Exit(2)
			}
			stop = true
		}
		if stop || missing {
			break
		}
	}
	if missing {
		fmt.Fprintf(Stderr, "Error: Missing argument for '%s' option\n", args[i])
		os.Exit(3)
	}
	if i < length {
		fmt.Fprintf(Stderr, "Error: Extranous command-line argument '%s'\n", args[i])
		os.Exit(4)
	}
}

func main() {
	parseArgs(os.Args)

	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.CoreNamespace)
	InitInternalLibs()

	envForNs := map[string]struct{}{}

	for _, f := range CoreSourceFiles {
		GLOBAL_ENV.SetCurrentNamespace(GLOBAL_ENV.CoreNamespace)
		nsName := CoreNameAsNamespaceName(f.Name)
		nsNamePtr := STRINGS.Intern(nsName)

		if ns, found := GLOBAL_ENV.Namespaces[nsNamePtr]; found {
			if _, exists := envForNs[nsName]; exists {
				continue // Already processed; this is probably a linter*.joke file
			}
			if Verbose > 0 {
				fmt.Printf("FOUND ns=%s file=%s mappings=%d\n", nsName, f.Filename, len(ns.Mappings()))
			}
		} else {
			if Verbose > 0 {
				fmt.Printf("READING ns=%s file=%s\n", nsName, f.Filename)
			}
		}

		envForNs[nsName] = struct{}{}

		ProcessCoreSourceFileFor(f.Name)

		ns := GLOBAL_ENV.Namespaces[nsNamePtr]
		if Verbose > 0 {
			fmt.Printf("READ ns=%s mappings=%d\n", nsName, len(ns.Mappings()))
		}
	}

	if Verbose > 1 {
		fmt.Fprintln(Stderr, "gen_code:main(): After loading source files:")
		Spew()
	}

	statics := []string{}
	runtime := []string{}
	imports := NewImports()

	// Mark "everything" as used.
	ResetUsage()

	genEnv := &GenEnv{
		Statics:        &statics,
		StaticImport:   imports,
		Runtime:        &runtime,
		Required:       nil,
		Import:         imports,
		Namespace:      GLOBAL_ENV.CoreNamespace,
		Runtimes:       map[*Namespace]*[]string{},
		Imports:        map[*Namespace]*Imports{},
		Requireds:      map[*Namespace]*map[*Namespace]struct{}{},
		Generated:      map[interface{}]interface{}{},
		CoreNamespaces: envForNs,
		LateInit:       false,
	}

	genEnv.emitVar("STR", STR)
	genEnv.emitVar("STRINGS", STRINGS)
	genEnv.emitVar("SYMBOLS", SYMBOLS)
	genEnv.emitVar("SPECIAL_SYMBOLS", SPECIAL_SYMBOLS)
	genEnv.emitVar("KEYWORDS", KEYWORDS)
	genEnv.emitVar("TYPE", TYPE)
	genEnv.emitVar("TYPES", TYPES)
	genEnv.emitVar("GLOBAL_ENV", GLOBAL_ENV)

	/* Emit the per-namespace files (a_*_code.go). */

	for nsNamePtr, ns := range GLOBAL_ENV.Namespaces {
		const fileTemplate = `
// Generated by gen_code. Don't modify manually!

// +build fast_init

package core

import (
{imports}
)

func init() {
	{name}NamespaceInfo = internalNamespaceInfo{available: true}
{lazy}
{static}
}

func {name}Init() {
{runtime}
}
`

		nsName := *nsNamePtr

		GLOBAL_ENV.SetCurrentNamespace(ns)

		if _, found := envForNs[nsName]; !found {
			if Verbose > 0 {
				fmt.Printf("LAZILY INITIALIZING ns=%s mappings=%d\n", nsName, len(ns.Mappings()))
			}
			continue
		}

		if Verbose > 0 {
			fmt.Printf("OUTPUTTING ns=%s mappings=%d\n", nsName, len(ns.Mappings()))
		}

		runtime := *genEnv.Runtimes[ns]
		if requireds, yes := genEnv.Requireds[ns]; yes && requireds != nil {
			for r, _ := range *requireds {
				rqNsName := r.ToString(false)
				filename := CoreSourceFilename[rqNsName]
				if filename == "" {
					filename = "<std>"
				}
				if Verbose > 0 {
					fmt.Printf("  REQUIRES: %s from %s\n", rqNsName, filename)
				}
				runtime = append(runtime, fmt.Sprintf(`
	ns_%s.MaybeLazy("%s")`[1:],
					NameAsGo(rqNsName), nsName))
			}
		}
		r := strings.Join(runtime, "\n")

		imports := genEnv.Imports[ns]
		imp := QuotedImportList(imports, "\n")
		if len(imp) > 0 && imp[0] == '\n' {
			imp = imp[1:]
		}

		filename := CoreSourceFilename[nsName]
		name := filename[0 : len(filename)-5] // assumes .joke extension
		goname := NameAsGo(nsName)
		codeFile := fmt.Sprintf(codePattern, name)
		fileContent := fileTemplate[1:]
		lazy := ""
		statics := ""
		if nsName != "joker.core" {
			lazy = fmt.Sprintf(`
	ns_{goname}.Lazy = %sInit
`[1:],
				name)
		} else {
			statics = r
			r = ""
		}
		var trPerNs = [][2]string{
			{"{name}", name},
			{"{lazy}", lazy},
			{"{goname}", goname},
			{"{imports}", imp},
			{"{static}", statics},
			{"{runtime}", r},
		}

		for _, t := range trPerNs {
			fileContent = strings.ReplaceAll(fileContent, t[0], t[1])
		}
		ioutil.WriteFile(codeFile, []byte(fileContent), 0666)
	}

	/* Output the master file (a_code.go). */

	if Verbose > 0 {
		fmt.Printf("OUTPUTTING %s\n", masterFile)
	}

	r := strings.Join(runtime, "\n")
	if r != "" {
		r = fmt.Sprintf(`

func init() {
%s
}`,
			r)

	}

	imp := QuotedImportList(genEnv.StaticImport, "\n")
	if len(imp) > 0 && imp[0] == '\n' {
		imp = imp[1:]
	}

	var tr = [][2]string{
		{"{imports}", imp},
		{"{statics}", strings.Join(statics, "\n")},
		{"{runtime}", r},
	}

	fileContent := `
// Generated by gen_code. Don't modify manually!

// +build fast_init

package core

import (
{imports}
)

{statics}
{runtime}
`[1:]

	for _, t := range tr {
		fileContent = strings.Replace(fileContent, t[0], t[1], 1)
	}

	ioutil.WriteFile(masterFile, []byte(fileContent), 0666)

}

func (genEnv *GenEnv) emitVar(name string, obj interface{}) {
	if _, found := genEnv.Generated[name]; found {
		return // panic(fmt.Sprintf("already generated %s", name))
	}
	genEnv.Generated[name] = nil
	v := reflect.ValueOf(obj)

	if strings.HasPrefix(name, "var_") {
		ns := v.Interface().(Var).Namespace()
		if _, found := genEnv.CoreNamespaces[ns.ToString(false)]; found {
			oldNamespace := genEnv.Namespace
			oldRuntime := genEnv.Runtime
			oldImports := genEnv.Import
			oldRequired := genEnv.Required
			defer func() {
				genEnv.Namespace = oldNamespace
				genEnv.Runtime = oldRuntime
				genEnv.Import = oldImports
				genEnv.Required = oldRequired
			}()

			genEnv.Namespace = ns

			rt, found := genEnv.Runtimes[ns]
			if !found {
				newRuntime := []string{}
				rt = &newRuntime
				genEnv.Runtimes[ns] = rt
			}
			genEnv.Runtime = rt

			imp, found := genEnv.Imports[ns]
			if !found {
				newImport := Imports{}
				imp = &newImport
				genEnv.Imports[ns] = imp
			}
			genEnv.Import = imp

			rq, found := genEnv.Requireds[ns]
			if !found {
				newRequired := map[*Namespace]struct{}{}
				rq = &newRequired
				genEnv.Requireds[ns] = rq
			}
			genEnv.Required = rq
		}
	}

	*genEnv.Statics = append(*genEnv.Statics, fmt.Sprintf(`
var %s %s = %s`[1:],
		name, coreTypeName(v), genEnv.emitValue(name, reflect.TypeOf(nil), v)))
	genEnv.Generated[name] = obj
}

func (genEnv *GenEnv) emitMembers(target string, name string, obj interface{}) (members []string) {
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
			k := genEnv.emitValue("", reflect.TypeOf(nil), key)
			vi := v.MapIndex(key)
			v := genEnv.emitValue(fmt.Sprintf("%s[%s]%s", target, k, assertValueType(target, k, valueType, vi)), valueType, vi)
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
			val := genEnv.emitValue(fmt.Sprintf("%s.%s%s", target, vtf.Name, assertValueType(target, vtf.Name, vtf.Type, vf)), vtf.Type, vf)
			if val == "" {
				continue
			}
			members = append(members, fmt.Sprintf(`
	%s: %s,`[1:],
				vtf.Name, val))
		}
	default:
		panic(fmt.Sprintf("unsupported type %T for %s", obj, name))
	}
	sort.Strings(members)
	return
}

func (genEnv *GenEnv) emitValue(target string, t reflect.Type, v reflect.Value) string {
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
			importedAs := AddImport(genEnv.StaticImport, "", components[0], true)
			t = fmt.Sprintf("%s.%s", importedAs, components[1])
		}
		el := ""
		if t[0] != '*' {
			t = "*" + t
			el = ".Elem()"
		}
		importedAs := AddImport(genEnv.StaticImport, "", "reflect", true)
		return fmt.Sprintf("%s.TypeOf((%s)(nil))%s", importedAs, t, el)
	case "core":
	case ".":
	default:
		panic(fmt.Sprintf("unexpected PkgPath `%s' for %+v", pkg, v.Interface()))
	}

	switch v.Kind() {
	case reflect.Interface:
		return genEnv.emitValue(target, t, v.Elem())

	case reflect.Ptr:
		return genEnv.emitPtrTo(target, v)

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
		return fmt.Sprintf(`%s{%s}`, coreTypeName(v), genEnv.emitSlice(target, v))

	case reflect.Struct:
		typeName := coreTypeName(v)
		obj := v.Interface()
		lazy := ""
		switch obj := obj.(type) {
		case Proc:
			return genEnv.emitProc(target, obj)
		case Namespace:
			nsName := obj.Name.Name()
			if Verbose > 0 {
				fmt.Printf("COMPILING %s\n", nsName)
			}
			if nsName != "joker.core" {
				lateInit := genEnv.LateInit
				defer func() { genEnv.LateInit = lateInit }()
				genEnv.LateInit = true
			}
		case VarRefExpr:
			if curRequired := genEnv.Required; curRequired != nil {
				if vr := obj.Var(); vr != nil {
					if ns := vr.Namespace(); ns != nil && ns != genEnv.Namespace && ns != GLOBAL_ENV.CoreNamespace {
						(*curRequired)[ns] = struct{}{}
					}
				}
			}
		}
		if obj == nil {
			return ""
		}
		members := genEnv.emitMembers(target, typeName, obj)
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
			typeName, joinMembers(genEnv.emitMembers(target, typeName, obj)))

	default:
		return fmt.Sprintf("nil /* UNKNOWN TYPE obj=%T v=%s v.Kind()=%s vt=%s */", v.Interface(), v, v.Kind(), v.Type())
	}
}

func stdPackageName(pkg string) string {
	if strings.HasPrefix(pkg, "std/") {
		pkg = pkg[4:]
	}
	return pkg
}

func (genEnv *GenEnv) emitProc(target string, p Proc) string {
	fnName := NameAsGo(p.Name)
	newPackage := ""
	if p.Package != "" {
		pkgName := stdPackageName(p.Package)
		thunkName := fmt.Sprintf("STD_thunk_%s_%s", NameAsGo(pkgName), fnName)
		*genEnv.Statics = append(*genEnv.Statics, fmt.Sprintf(`
// std/%s/a_%s_fast_init.go's init() function sets this to the same as its local var %s on the !fast_init side:
var %s_var ProcFn
func %s(a []Object) Object {
	return %s_var(a)
}`[1:],
			pkgName, pkgName, fnName, thunkName, thunkName, thunkName))
		newPackage = fmt.Sprintf(`
	Package: %s,
`[1:],
			strconv.Quote(p.Package))
		fnName = thunkName
	}
	return fmt.Sprintf(`
Proc{
	Fn: %s,
	Name: %s,
%s}`[1:],
		fnName, strconv.Quote(fnName), newPackage)
}

func (genEnv *GenEnv) emitPtrToRegexp(target string, v reflect.Value) string {
	importedAs := AddImport(genEnv.Import, "", "regexp", true)
	source := fmt.Sprintf("%s.MustCompile(%s)", importedAs, strconv.Quote(v.Interface().(*regexp.Regexp).String()))
	*genEnv.Runtime = append(*genEnv.Runtime, fmt.Sprintf(`
	%s = %s`[1:],
		asTarget(target), source))
	return fmt.Sprintf("nil /* %s: &%s */", genEnv.Namespace.ToString(false), source)
}

func (genEnv *GenEnv) emitPtrTo(target string, ptr reflect.Value) string {
	if ptr.IsNil() {
		return "nil"
	}

	v := ptr.Elem()
	v = UnsafeReflectValue(v)

	switch pkg := path.Base(v.Type().PkgPath()); pkg {
	case "regexp":
		return genEnv.emitPtrToRegexp(target, ptr)
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
		return genEnv.emitPtrTo(target, v.Elem())

	default:
		thing, found := genEnv.Generated[v]
		if !found {
			ptrToObj := ptr.Interface()
			obj := v.Interface()
			name := uniqueId(ptrToObj)
			if genEnv.LateInit {
				if destVar, yes := ptrToObj.(*Var); yes {
					if e, isVarRefExpr := destVar.Expr().(*VarRefExpr); isVarRefExpr {
						sourceVarName := e.Var().Name()
						if _, found := knownLateInits[sourceVarName]; found {
							destVarId := uniqueId(destVar)
							*genEnv.Runtime = append(*genEnv.Runtime, fmt.Sprintf(`
	%s.Value = %s.Value`[1:],
								destVarId, uniqueId(e.Var())))
						}
					}
				}
			}
			genEnv.Generated[v] = name
			genEnv.emitVar(name, obj)
			return "&" + name
		}
		name := thing.(string)
		status, found := genEnv.Generated[name]
		if !found {
			panic(fmt.Sprintf("cannot find generated thing %s: %+v", name, v.Interface()))
		}
		if status == nil {
			*genEnv.Runtime = append(*genEnv.Runtime, fmt.Sprintf(`
	%s = &%s`[1:],
				asTarget(target), name))
			return fmt.Sprintf("nil /* %s: &%s */", genEnv.Namespace.ToString(false), name)
		}
		return "&" + name
	}
}

func (genEnv *GenEnv) emitSlice(target string, v reflect.Value) string {
	numEntries := v.Len()
	elemType := v.Type().Elem()
	el := []string{}
	for i := 0; i < numEntries; i++ {
		res := genEnv.emitValue(fmt.Sprintf("%s[%d]", target, i), elemType, v.Index(i))
		if res == "" {
			el = append(el, "\tnil,")
		} else {
			el = append(el, "\t"+res+",")
		}
	}
	return joinMembers(el)
}

func coreTypeString(s string) string {
	return strings.Replace(s, "core.", "", 1)
}

func coreTypeName(v reflect.Value) string {
	return coreTypeString(v.Type().String())
}

func coreTypeOf(obj interface{}) string {
	return coreTypeName(reflect.ValueOf(obj))
}

func uniqueId(obj interface{}) string {
	switch obj := obj.(type) {
	case *string:
		return "s_" + NameAsGo(*obj)
	default:
	}
	return UniqueId(obj)
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
	return ".(" + coreTypeName(r) + ")"
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

func asTarget(s string) string {
	if s == "" || s[len(s)-1] != ')' {
		return s
	}
	ix := strings.LastIndex(s, ".(")
	if ix < 0 {
		return s
	}
	return s[0:ix]
}

/* Represents an 'import ( foo "bar/bletch/foo" )' line to be produced. */
type Import struct {
	Local       string // "foo", "_", ".", or empty
	LocalRef    string // local unless empty, in which case final component of full (e.g. "foo")
	Full        string // "bar/bletch/foo"
	substituted bool   // Had to substitute a different local name
}

/* Maps relative package (unix-style) names to their imports, non-emptiness, etc. */
type Imports struct {
	LocalNames map[string]string  // "foo" -> "bar/bletch/foo"; no "_" nor "." entries here
	FullNames  map[string]*Import // "bar/bletch/foo" -> ["foo", "bar/bletch/foo"]
}

func NewImports() *Imports {
	return &Imports{map[string]string{}, map[string]*Import{}}
}

/* Given desired local and the full (though relative) name of the
/* package, make sure the local name agrees with any existing entry
/* and isn't already used (picking an alternate local name if
/* necessary), add the mapping if necessary, and return the (possibly
/* alternate) local name. */
func AddImport(imports *Imports, local, full string, okToSubstitute bool) string {
	components := strings.Split(full, "/")
	if imports == nil {
		panic(fmt.Sprintf("imports is nil for %s", full))
	}
	if e, found := imports.FullNames[full]; found {
		if e.Local == local {
			return e.LocalRef
		}
		if okToSubstitute {
			return e.LocalRef
		}
		panic(fmt.Sprintf("addImport(%s,%s) told to to replace (%s,%s)", local, full, e.Local, e.Full))
	}

	substituted := false
	localRef := local
	if local == "" {
		localRef = components[len(components)-1]
	}
	if localRef != "." {
		prevComponentIndex := len(components) - 1
		for {
			origLocalRef := localRef
			curFull, found := imports.LocalNames[localRef]
			if !found {
				break
			}
			substituted = true
			prevComponentIndex--
			if prevComponentIndex >= 0 {
				localRef = components[prevComponentIndex] + "_" + localRef
				continue
			} else if prevComponentIndex > -99 /* avoid infinite loop */ {
				localRef = fmt.Sprintf("%s_%d", origLocalRef, -prevComponentIndex)
				continue
			}
			panic(fmt.Sprintf("addImport(%s,%s) trying to replace (%s,%s)", localRef, full, imports.FullNames[curFull].LocalRef, curFull))
		}
		if imports.LocalNames == nil {
			imports.LocalNames = map[string]string{}
		}
		imports.LocalNames[localRef] = full
	}
	if imports.FullNames == nil {
		imports.FullNames = map[string]*Import{}
	}
	imports.FullNames[full] = &Import{local, localRef, full, substituted}
	return localRef
}

func sortedImports(pi *Imports, f func(k string, v *Import)) {
	var keys []string
	for k, _ := range pi.FullNames {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := pi.FullNames[k]
		f(k, v)
	}
}

func QuotedImportList(pi *Imports, prefix string) string {
	imports := ""
	sortedImports(pi,
		func(k string, v *Import) {
			if (v.Local == "" && !v.substituted) || v.Local == path.Base(k) {
				imports += prefix + `"` + k + `"`
			} else {
				imports += prefix + v.LocalRef + ` "` + k + `"`
			}
		})
	return imports
}

// NOTE: Below this line, code comes from github.com/jcburley/go-spew:

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
	if canSortSimply(vs.values[0].Kind()) {
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
func canSortSimply(kind reflect.Kind) bool {
	// This switch parallels valueSortLess, except for the default case.
	switch kind {
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
		return a.String() < b.String()
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
