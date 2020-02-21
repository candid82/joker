package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	. "github.com/candid82/joker/core"
	. "github.com/candid82/joker/core/gen_common"
	"github.com/candid82/joker/core/gen_go"
	_ "github.com/candid82/joker/std/html"
	_ "github.com/candid82/joker/std/string"
)

func parseArgs(args []string) {
	length := len(args)
	stop := false
	missing := false
	var i int
	for i = 1; i < length; i++ { // shift
		switch args[i] {
		case "--verbose":
			VerbosityLevel++
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

const hextable = "0123456789abcdef"
const masterFile = "a_code.go"
const codePattern = "a_%s_code.go"
const dataPattern = "a_%s_data.go"

type GenEnv struct {
	GenGo            *gen_go.GenGo
	StaticImport     *Imports
	Import           *Imports
	Namespace        *Namespace
	Required         *map[*Namespace]struct{} // Namespaces referenced by current one
	Runtimes         map[*Namespace]*[]string
	Imports          map[*Namespace]*Imports
	Namespaces       map[string]int                          // Set of the known namespaces (core, user, required stds)
	NamespaceIndices map[string]int                          // Order of "discovery" for known namespaces
	Requireds        map[*Namespace]*map[*Namespace]struct{} // Namespaces referenced by each namespace
	LateInit         bool                                    // Whether emitting a namespace other than joker.core
}

/* NewEnv() statically declares numerous variables and sets some of
/* them to initial values before any func init()'s run. Later, func
/* main() calls (*Env)InitEnv() and other receivers, as appropriate,
/* to set per-invocation values. Normally, by this point in time,
/* core.joke (in digested form) has already been "run", so it cannot
/* directly use such values; but the other *.joke files won't have
/* been run yet, so can use those values. However, that won't work
/* here (in gen_code), since there's no corresponding main.go and (of
/* course) the per-invocation-of-Joker values won't be known. So
/* e.g. `(def *foo* *out*)`, while it'd work normally, will end up
/* setting `*foo*` to nil; and (*Env)InitEnv() won't know to fix that
/* up after setting `*out*` to a non-nil value. This map lists these
/* "late-initialization" variables so code to set them is emitted in
/* the respective namespace's func *Init() code. Any variable defined
/* in terms of one of these variables is thus handled and, in turn,
/* added to this map. Currently, only direct assignment is handled;
/* something like `(def *n* (count *command-line-args*))` would thus
/* not work without special handling, which does not yet appear to be
/* necessary. */
var knownLateInits = map[string]struct{}{
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

func runtimeSortFunc(runtime []string, i, j int) bool {
	iStrings := strings.SplitN(runtime[i], " = ", 2)
	jStrings := strings.SplitN(runtime[j], " = ", 2)
	if len(iStrings) != len(jStrings) {
		return len(iStrings) > len(jStrings)
	}
	if len(iStrings) == 1 || iStrings[1] == jStrings[1] {
		return iStrings[0] < jStrings[0]
	}
	return iStrings[1] < jStrings[1]
}

func main() {
	parseArgs(os.Args)

	coreSourceFilename := map[string]string{}
	namespaceIndex := 0
	var namespaces = map[string]int{}

	for _, f := range CoreSourceFiles {
		GLOBAL_ENV.SetCurrentNamespace(GLOBAL_ENV.CoreNamespace)
		nsName := CoreNameAsNamespaceName(f.Name)
		nsNamePtr := STRINGS.Intern(nsName)

		if ns, found := GLOBAL_ENV.Namespaces[nsNamePtr]; found {
			if _, exists := namespaces[nsName]; exists {
				continue // Already processed; this is probably a linter*.joke file
			}
			if VerbosityLevel > 0 {
				fmt.Printf("FOUND ns=%s file=%s mappings=%d\n", nsName, f.Filename, len(ns.Mappings()))
			}
		} else {
			if VerbosityLevel > 0 {
				fmt.Printf("READING ns=%s file=%s\n", nsName, f.Filename)
			}
		}

		coreSourceFilename[nsName] = f.Filename
		namespaces[nsName] = namespaceIndex
		namespaceIndex++

		ns := GLOBAL_ENV.Namespaces[nsNamePtr]

		ns.MaybeLazy("gen_code") // Process the namespace (read the digested data for e.g. joker.core)

		if VerbosityLevel > 0 {
			fmt.Printf("READ ns=%s mappings=%d\n", nsName, len(ns.Mappings()))
		}
	}

	statics := []string{}
	runtime := []string{}
	imports := NewImports()

	// Mark "everything" as used.
	ResetUsage()

	genGo := &gen_go.GenGo{
		Statics:        &statics,
		Runtime:        &runtime,
		Generated:      map[interface{}]interface{}{},
		TypeToStringFn: coreTypeString,
		FieldSortFn:    sort.Strings,
	}
	genEnv := &GenEnv{
		GenGo:            genGo,
		StaticImport:     imports,
		Required:         nil,
		Import:           imports,
		Namespace:        GLOBAL_ENV.CoreNamespace,
		Runtimes:         map[*Namespace]*[]string{},
		Imports:          map[*Namespace]*Imports{},
		Requireds:        map[*Namespace]*map[*Namespace]struct{}{},
		Namespaces:       namespaces,
		NamespaceIndices: map[string]int{},
		LateInit:         false,
	}

	genGo.KeySortFn = func(genEnv *GenEnv) func([]reflect.Value) {
		return func(values []reflect.Value) {
			genEnv.sortValues(values)
		}
	}(genEnv)
	genGo.WhereFn = func(genEnv *GenEnv) func() string {
		return func() string {
			return genEnv.Namespace.ToString(false)
		}
	}(genEnv)
	genGo.StructHookFn = func(genEnv *GenEnv) func(target string, t reflect.Type, obj interface{}) (res string, deferredFunc func(target string, obj interface{})) {
		return func(target string, t reflect.Type, obj interface{}) (res string, deferredFunc func(target string, obj interface{})) {
			return genEnv.structHookFn(target, obj)
		}
	}(genEnv)
	genGo.ValueHookFn = func(genEnv *GenEnv) func(target string, t reflect.Type, v reflect.Value) string {
		return func(target string, t reflect.Type, v reflect.Value) string {
			return genEnv.valueHookFn(target, t, v)
		}
	}(genEnv)
	genGo.PointerHookFn = func(genEnv *GenEnv) func(target string, ptr, v reflect.Value) string {
		return func(target string, ptr, v reflect.Value) string {
			return genEnv.pointerHookFn(target, ptr, v)
		}
	}(genEnv)
	genGo.PtrToValueFn = func(genEnv *GenEnv) func(ptr, v reflect.Value) string {
		return func(ptr, v reflect.Value) string {
			return genEnv.ptrToValueFn(ptr, v)
		}
	}(genEnv)

	// Order namespaces by when "discovered" for stability
	// compiling and outputting.  Put the non-core namespaces
	// (user and std namespaces required by core) in front, so
	// they are emitted first.

	totalNs := len(GLOBAL_ENV.Namespaces)
	coreNs := namespaceIndex
	coreBaseIndex := totalNs - coreNs

	namespaceArray := make([]string, totalNs)

	{
		otherNs := []string{}
		for nsNamePtr, _ := range GLOBAL_ENV.Namespaces {
			nsName := *nsNamePtr
			index, found := genEnv.Namespaces[nsName]
			if !found {
				// A std lib or user; stabilize the order of these later.
				otherNs = append(otherNs, nsName)
				continue
			}
			namespaceArray[coreBaseIndex+index] = nsName
			genEnv.NamespaceIndices[nsName] = index
		}
		sort.Strings(otherNs) // Stabilize the order of std libs and user.
		namespaceIndex = -1
		for _, nsName := range otherNs {
			index := namespaceIndex
			namespaceIndex--
			namespaceArray[coreBaseIndex+index] = nsName
			genEnv.NamespaceIndices[nsName] = index
		}
	}

	GLOBAL_ENV.ReferCoreToUser()

	// Emit the "global" (static) Joker variables.

	genGo.Var("STR", false, STR)
	genGo.Var("STRINGS", false, STRINGS)
	genGo.Var("SYMBOLS", false, SYMBOLS)
	genGo.Var("SPECIAL_SYMBOLS", false, SPECIAL_SYMBOLS)
	genGo.Var("KEYWORDS", false, KEYWORDS)
	genGo.Var("TYPE", false, TYPE)
	genGo.Var("TYPES", false, TYPES)
	genGo.Var("GLOBAL_ENV", true, GLOBAL_ENV) // init var at runtime to avoid cycles

	// Emit the per-namespace files (a_*_code.go).

	for _, nsName := range namespaceArray {
		ns := GLOBAL_ENV.Namespaces[STRINGS.Intern(nsName)]

		const fileTemplate = `
// Generated by gen_code. Don't modify manually!

// +build !gen_data
// +build fast_init

package core

import (
{imports}
)

func init() {
{lazy}
{static}
}
{runtime}
`

		GLOBAL_ENV.SetCurrentNamespace(ns)

		if _, found := genEnv.Namespaces[nsName]; !found {
			if VerbosityLevel > 0 {
				fmt.Printf("DIRECTLY INITIALIZING ns=%s mappings=%d in %s\n", nsName, len(ns.Mappings()), masterFile)
			}
			if _, found := genEnv.Runtimes[ns]; found {
				msg := fmt.Sprintf("ERROR: found runtime info for ns=%s", nsName)
				panic(msg)
			}
			continue
		}

		var name string
		if filename, found := coreSourceFilename[nsName]; found {
			name = filename[0 : len(filename)-5] // assumes .joke extension
		} else {
			// This shouldn't happen, but pick reasonable names in case it's enabled for some reason.
			filename = name + ".joke"
			name = strings.ReplaceAll(nsName, "joker.", "")
		}
		goname := StringAsGoName(nsName)
		codeFile := fmt.Sprintf(codePattern, name)

		if VerbosityLevel > 0 {
			fmt.Printf("OUTPUTTING %s as %s (mappings=%d)\n", nsName, codeFile, len(ns.Mappings()))
		}

		var r string
		if runtimePtr, found := genEnv.Runtimes[ns]; found {
			runtime := *runtimePtr
			if requireds, yes := genEnv.Requireds[ns]; yes && requireds != nil {
				for r, _ := range *requireds {
					rqNsName := r.ToString(false)
					filename := coreSourceFilename[rqNsName]
					if filename == "" {
						filename = "<std>"
					}
					if VerbosityLevel > 0 {
						fmt.Printf("  REQUIRES: %s from %s\n", rqNsName, filename)
					}
					runtime = append(runtime, fmt.Sprintf(`
	ns_%s.MaybeLazy("%s")`[1:],
						StringAsGoName(rqNsName), nsName))
				}
			}
			sort.SliceStable(runtime, func(i, j int) bool {
				return runtimeSortFunc(runtime, i, j)
			})
			r = strings.Join(runtime, "\n")
		}

		var imp string
		if imports, found := genEnv.Imports[ns]; found {
			imp = QuotedImportList(imports, "\n")
			if len(imp) > 0 && imp[0] == '\n' {
				imp = imp[1:]
			}
		}

		fileContent := fileTemplate[1:]

		lazy := ""
		statics := ""
		if nsName == "joker.core" {
			statics = r
			r = ""
		} else {
			lazy = fmt.Sprintf(`
	ns_{goname}.Lazy = %sLazyInit
`[1:],
				name)
			r = fmt.Sprintf(`

func %sLazyInit() {
%s
}
`[1:],
				name, r)
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

	if VerbosityLevel > 0 {
		fmt.Printf("OUTPUTTING %s\n", masterFile)
	}

	sort.SliceStable(runtime, func(i, j int) bool {
		return runtimeSortFunc(runtime, i, j)
	})
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

func stdPackageName(pkg string) string {
	if strings.HasPrefix(pkg, "std/") {
		pkg = pkg[4:]
	}
	return pkg
}

func (genEnv *GenEnv) emitProc(target string, p Proc) string {
	fnName := StringAsGoName(p.Name)
	newPackage := ""
	if p.Package != "" {
		pkgName := stdPackageName(p.Package)
		thunkName := fmt.Sprintf("STD_thunk_%s_%s", StringAsGoName(pkgName), fnName)
		*genEnv.GenGo.Statics = append(*genEnv.GenGo.Statics, fmt.Sprintf(`
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
	*genEnv.GenGo.Runtime = append(*genEnv.GenGo.Runtime, fmt.Sprintf(`
	%s = %s`[1:],
		gen_go.AsTarget(target), source))
	return fmt.Sprintf("nil /* %s: &%s */", genEnv.Namespace.ToString(false), source)
}

func coreTypeString(s string) string {
	return strings.Replace(s, "core.", "", 1)
}

func uniqueId(obj interface{}) string {
	switch obj := obj.(type) {
	case *string:
		return "s_" + StringAsGoName(*obj)
	default:
	}
	return UniqueId(obj)
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

func (genEnv *GenEnv) substituteString(aStr string) string {
	if aIndex, found := genEnv.NamespaceIndices[aStr]; found {
		aStr = fmt.Sprintf("[%03d]%s", aIndex, aStr)
	}
	return aStr
}

func (genEnv *GenEnv) sortValues(keys []reflect.Value) {
	stringSubstitutionFn := func(genEnv *GenEnv) func(string) string {
		return genEnv.substituteString
	}(genEnv)
	gen_go.SortValues(keys, stringSubstitutionFn)
}

func (genEnv *GenEnv) structHookFn(target string, obj interface{}) (res string, deferredFunc func(target string, obj interface{})) {
	res = "-" // Means no result, continue processing
	switch obj := obj.(type) {
	case Proc:
		return genEnv.emitProc(target, obj), nil
	case Namespace:
		nsName := obj.Name.Name()
		if VerbosityLevel > 0 {
			fmt.Printf("COMPILING %s\n", nsName)
		}
		lateInit := genEnv.LateInit
		genEnv.LateInit = nsName != "joker.core"
		deferredFunc = func(target string, obj interface{}) {
			genEnv.LateInit = lateInit
			if VerbosityLevel > 0 {
				fmt.Printf("FINISHED %s\n", nsName)
			}
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
	return
}

func (genEnv *GenEnv) valueHookFn(target string, t reflect.Type, v reflect.Value) string {
	switch pkg := v.Type().PkgPath(); pkg {
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
	}

	switch pkg := path.Base(v.Type().PkgPath()); pkg {
	case "core":
	case ".":
	default:
		panic(fmt.Sprintf("unexpected PkgPath `%s' for %+v", pkg, v.Interface()))
	}
	return ""
}

func (genEnv *GenEnv) pointerHookFn(target string, ptr, v reflect.Value) string {
	switch pkg := v.Type().PkgPath(); pkg {
	case "regexp":
		return genEnv.emitPtrToRegexp(target, ptr)
	}

	switch pkg := path.Base(v.Type().PkgPath()); pkg {
	case "core":
	case ".":
	default:
		panic(fmt.Sprintf("unexpected PkgPath `%s' for &%+v", pkg, v.Interface()))
	}
	return ""
}

func (genEnv *GenEnv) ptrToValueFn(ptr, v reflect.Value) string {
	ptrToObj := ptr.Interface()
	obj := v.Interface()
	name := uniqueId(ptrToObj)
	if genEnv.LateInit {
		if destVar, yes := ptrToObj.(*Var); yes {
			if e, isVarRefExpr := destVar.Expr().(*VarRefExpr); isVarRefExpr {
				sourceVarName := e.Var().Name()
				if _, found := knownLateInits[sourceVarName]; found {
					destVarId := uniqueId(destVar)
					*genEnv.GenGo.Runtime = append(*genEnv.GenGo.Runtime, fmt.Sprintf(`
	%s.Value = %s.Value`[1:],
						destVarId, uniqueId(e.Var())))
				}
			}
		}
	}
	genEnv.GenGo.Generated[v] = name

	if ns, yes := ptrToObj.(*Namespace); yes {
		if _, found := genEnv.Namespaces[ns.ToString(false)]; found {
			oldNamespace := genEnv.Namespace
			oldRuntime := genEnv.GenGo.Runtime
			oldImports := genEnv.Import
			oldRequired := genEnv.Required
			defer func() {
				genEnv.Namespace = oldNamespace
				genEnv.GenGo.Runtime = oldRuntime
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
			genEnv.GenGo.Runtime = rt

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

	genEnv.GenGo.Var(name, false, obj)
	return "&" + name
}
