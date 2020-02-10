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
	GenGo      gen_go.GoGen
	Required   *map[*Namespace]struct{} // Namespaces referenced by current one
	Runtimes   map[*Namespace]*[]string
	Imports    map[*Namespace]*Imports
	Namespaces map[string]int                          // Set of the known namespaces (core, user, required stds)
	Requireds  map[*Namespace]*map[*Namespace]struct{} // Namespaces referenced by each namespace
	LateInit   bool                                    // Whether emitting a namespace other than joker.core
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

var namespaces = map[string]int{}
var namespaceIndices = map[string]int{}

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

	if VerbosityLevel > 1 {
		fmt.Fprintln(Stderr, "gen_code:main(): After loading source files:")
		Spew()
	}

	statics := []string{}
	runtime := []string{}
	imports := NewImports()

	// Mark "everything" as used.
	ResetUsage()

	goGen := &gen_go.GoGen{
		Statics:      &statics,
		StaticImport: imports,
		Runtime:      &runtime,
		Import:       imports,
		Generated:    map[interface{}]interface{}{},
	}
	genEnv := &GenEnv{
		GoGen:      &goGen,
		Required:   nil,
		Namespace:  GLOBAL_ENV.CoreNamespace,
		Runtimes:   map[*Namespace]*[]string{},
		Imports:    map[*Namespace]*Imports{},
		Requireds:  map[*Namespace]*map[*Namespace]struct{}{},
		Namespaces: namespaces,
		LateInit:   false,
	}

	// Order namespaces by when "discovered" for stability
	// compiling and outputting.  Put the non-core namespaces
	// (user and std namespaces required by core) in front, so
	// they are emitted first.

	totalNs := len(GLOBAL_ENV.Namespaces)
	coreNs := namespaceIndex
	coreBaseIndex := totalNs - coreNs

	namespaceArray := make([]string, totalNs)
	namespaceIndex = -1

	for nsNamePtr, _ := range GLOBAL_ENV.Namespaces {
		nsName := *nsNamePtr
		index, found := namespaces[nsName]
		if !found {
			// An std lib or user
			index = namespaceIndex
			namespaceIndex--
		}
		namespaceArray[coreBaseIndex+index] = nsName
		namespaceIndices[nsName] = index
	}

	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.CoreNamespace)

	// Emit the "global" (static) Joker variables.

	genEnv.emitVar("STR", false, STR)
	genEnv.emitVar("STRINGS", false, STRINGS)
	genEnv.emitVar("SYMBOLS", false, SYMBOLS)
	genEnv.emitVar("SPECIAL_SYMBOLS", false, SPECIAL_SYMBOLS)
	genEnv.emitVar("KEYWORDS", false, KEYWORDS)
	genEnv.emitVar("TYPE", false, TYPE)
	genEnv.emitVar("TYPES", false, TYPES)
	genEnv.emitVar("GLOBAL_ENV", true, GLOBAL_ENV) // init var at runtime to avoid cycles

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

		if _, found := namespaces[nsName]; !found {
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
		goname := NameAsGo(nsName)
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
						NameAsGo(rqNsName), nsName))
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

func (env *Env) ReferCoreToUser() {
	// Nothing need be done; it's already "baked in" in the fast-startup version.
}
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
			if bIndex, found := namespaceIndices[bStr]; found {
				return aIndex < bIndex
			}
		}
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
