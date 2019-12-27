package core

import (
	"encoding/binary"
	"fmt"
	"github.com/jcburley/go-spew/spew"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

// Order of runtime initialization:
//
// 1. All "static" initializers ("var ..." at file level). In Go, this
// can include (essentially) full-blown function calls.
//
// 2. init() functions, evidently in alphabetical order by filename
// (unsure of what happens when different directories are involved).
//
// 3. *Init() functions, in the order they're called (by procs.go).
//
// #2, above, is implemented by ensuring the "master code file",
// a_code.go, is named such that it comes before any other a_*_code.go
// file currently being generated. (gen_code.go should catch if this
// changes.)
//
// a_code.go initializes things that are not tied to namespaces,
// including strings and keywords (though the latter could be tied to
// namespaces, they aren't in current data/*.joke files).
//
// Within a_code.go's init() function:
//
// /* 00 */ interns existing (constant pointers to) strings and
// initializes (runtime pointers to) strings from the values as
// interned during initialization of mainline Joker code (via static
// initializers). Strings interned via init() routines likely won't
// work properly, as they'll be seen as "base" strings when
// gen_code.go runs, but might not be interned when a_code.go init()
// runs; currently, there don't seem to be any such strings.
//
// /* 01 */ initializes fields that need pointers to strings. This
// includes Keyword .name and .hash fields.
//
// Within an a_*_code.go's *Init() function:
//
// /* 00 */ initializes the local _ns variable to point to the current
// namespace.
//
// /* 01 */ initializes fields that need the value of _ns or any
// (runtime pointers to) strings that a_code.go's init() function
// initializes. (Constant pointers to strings are initialized via
// static fields.) It also initializes fields that need fully
// initialized keywords (handled by a_code.go's init() function) and
// Symbols (instantiated inline). Finally, it initializes fields that
// need TYPES[] of (pointers to) strings; this must wait until runtime
// so TYPES[] itself is fully initialized, so it might as well wait
// until all the (pointers to) strings have been initialized.
//
// /* 02 */ interns existing variables (which must, of course, be
// sufficiently populated at this point).
//
// /* 03 */ is where circular references (such as List .rest fields
// pointing back to a parent List) are initialized, and where deferred
// Object sequence members are initialized.
//
// /* 04 */ updates base variables with fully-initialized
// information. E.g. var_foo is used as a complete template for the
// base variable 'foo'; _ns.UpdateVar(..., var_foo) is called at this
// level, with the resulting pointer stored in p_var_foo (which does
// not point to var_foo, as that was just a template).  This
// implements (mainly) the (add-doc-and-meta ...) calls in core.joke.
//
// /* 05 */ Copies the p_var_* values updated in 04 into appropriate
// fields.

type (
	CodeEnv struct {
		CodeWriterEnv *CodeWriterEnv
		Namespace     *Namespace
		BaseMappings  map[*string]*Var
		Statics       string
		Interns       string
		Runtime       []func() string
		Need          map[string]Finisher
		Generated     map[interface{}]interface{} // nil: being generated; else: fully generated (self)
	}

	CodeWriterEnv struct {
		BaseStrings StringPool
		Need        map[string]Finisher
		Generated   map[interface{}]interface{} // nil: being generated; else: fully generated (self)
	}

	Finisher interface {
		Finish(name string, codeEnv *CodeEnv) string
	}

	NativeString struct {
		s string
	}

	InternedString struct {
		s string
	}
)

func (s NativeString) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := "s_" + NameAsGo(s.s)
	if _, ok := env.CodeWriterEnv.Need[name]; !ok {
		env.CodeWriterEnv.Need[name] = s // Don't overwrite an InternedString{}
	}
	return "!" + name
}

func (s NativeString) Finish(name string, env *CodeEnv) string {
	return ""
}

func (s InternedString) Finish(name string, env *CodeEnv) string {
	return ""
}

var tr = [][2]string{
	{"_", "US"},
	{"?", "Q"},
	{"!", "BANG"},
	{"<=", "LE"},
	{">=", "GE"},
	{"<", "LT"},
	{">", "GT"},
	{"=", "EQ"},
	{"'", "APOS"},
	{"+", "PLUS"},
	{"-", "DASH"},
	{"*", "STAR"},
	{"/", "SLASH"},
	{"&", "AMP"},
	{"#", "HASH"},
	{".", "DOT"},
	{"%", "PCT"},
	{".", "DOT"},
	{":", "COLON"},
}

func NameAsGo(name string) string {
	for _, t := range tr {
		name = strings.ReplaceAll(name, t[0], "_"+t[1]+"_")
	}
	return name
}

func noBang(s string) string {
	if len(s) > 0 && s[0] == '!' {
		return s[1:]
	}
	return s
}

func immediate(s string) (bool, string) {
	if len(s) > 0 && s[0] == '!' {
		return true, s[1:]
	}
	return false, s
}

func joinStatics(fields []string) string {
	f := strings.Join(fields, "\n")
	if f != "" {
		f = "\n" + f + "\n"
	}
	return f
}

func indirect(s string) string {
	if s[0] == '&' {
		return s[1:]
	}
	if s[0] == '!' || !notNil(s) {
		return s
	}
	return "*" + s
}

func notNil(s string) bool {
	return s != "" && s != "nil" && !strings.HasSuffix(s, "{}")
}

func ptrTo(s string) string {
	if !notNil(s) {
		return "nil"
	}
	if s[0] == '!' {
		s = s[1:]
	}
	if s[0] == '&' {
		return s
	}
	if strings.HasPrefix(s, "p_") || strings.HasPrefix(s, "STRINGS.Intern(") {
		return s
	}
	return "&" + s
}

func symAsGo(sym Symbol) string {
	name := "_EMPTY_"
	if sym.name != nil {
		name = NameAsGo(strings.ReplaceAll(sym.ToString(false), "/", "_FW_"))
	}
	if sym.info == nil {
		return name
	}
	return fmt.Sprintf("%s_%d_%d__%d_%d", name, sym.info.startLine, sym.info.startColumn, sym.info.endLine, sym.info.endColumn)
}

func (sym Symbol) AsGo() string {
	return "symbol_" + symAsGo(sym)
}

func kwAsGo(kw Keyword) string {
	name := NameAsGo(strings.ReplaceAll(strings.ReplaceAll(kw.ToString(false), "/", "_FW_"), ":", ""))
	if kw.info == nil {
		return name
	}
	return fmt.Sprintf("%s_%d_%d__%d_%d", name, kw.info.startLine, kw.info.startColumn, kw.info.endLine, kw.info.endColumn)
}

func (kw Keyword) AsGo() string {
	if kw.name != nil {
		return "keyword_" + kwAsGo(kw)
	}
	panic("empty keyword")
}

func (v Var) AsGo() string {
	name := symAsGo(v.name)
	if v.ns != nil {
		if v.name.ns != nil && *v.name.ns != *v.ns.Name.name {
			panic(fmt.Sprintf("Symbol namespace discrepancy: Var %s has %s, its sym has %s", name, *v.ns.Name.name, *v.name.ns))
		}
	}
	return "var_" + name
}

func (v VarRefExpr) AsGo() string {
	s := v.vr.AsGo()
	return fmt.Sprintf("%s_%d_%d", strings.Replace(s, "var_", "varRefExpr_", 1), v.startLine, v.startColumn)
}

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

func infoHolderNameAsGo(obj interface{}) (string, bool) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", false
	}
	vt := v.Type()
	sf, yes := vt.FieldByName("InfoHolder")
	if yes {
		if !sf.Anonymous {
			return "", false
		}
		v = v.FieldByName("InfoHolder")
		vt = v.Type()
		if vt.Kind() != reflect.Struct {
			return "", false
		}
		sf, yes = vt.FieldByName("info")
		if !yes || sf.Anonymous {
			return "", false
		}
		v = v.FieldByName("info")
		vt = v.Type()
		if vt.Kind() != reflect.Ptr {
			panic("'info' field not a pointer")
		}
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
		vt = v.Type()
	}
	sf, yes = vt.FieldByName("Position")
	if !yes || !sf.Anonymous {
		return "", false
	}
	v = v.FieldByName("Position")
	vt = v.Type()
	if vt.Kind() != reflect.Struct {
		return "", false
	}
	sf, yes = vt.FieldByName("startLine")
	if !yes || sf.Anonymous {
		return "", false
	}
	filenamePtr := UnsafeReflectValue(v.FieldByName("filename"))
	if filenamePtr.IsZero() || filenamePtr.IsNil() {
		return "", false
	}
	filename := filenamePtr.Elem().Interface().(string)
	if filename != "<joker.core>" { // TODO: Support other namespaces
		return "", false
	}
	startLine := UnsafeReflectValue(v.FieldByName("startLine")).Interface().(int)
	startColumn := UnsafeReflectValue(v.FieldByName("startColumn")).Interface().(int)
	endLine := UnsafeReflectValue(v.FieldByName("endLine")).Interface().(int)
	endColumn := UnsafeReflectValue(v.FieldByName("endColumn")).Interface().(int)
	return fmt.Sprintf("%d_%d__%d_%d", startLine, startColumn, endLine, endColumn), true
}

func UniqueId(obj, actual interface{}) (id string) {
	defer func() {
		if r := recover(); r != nil {
			id = coreTypeAsGo(obj)
			pos, havePos := infoHolderNameAsGo(obj)
			if havePos {
				id = id + "_" + pos
			}
			h := getHash()
			h.Write(([]byte)(spewConfig.Sdump(obj)))
			if reflect.ValueOf(obj).Kind() == reflect.Ptr {
				if actual == nil {
					actual = obj
				}
				id = fmt.Sprintf("%s_%p_%d", id, actual, h.Sum32())
			} else {
				id = fmt.Sprintf("%s_%d", id, h.Sum32())
			}
			origType := reflect.TypeOf(obj).String()
			if origType == "core.Keyword" || origType == "core.Symbol" {
				fmt.Printf("UniqueId: Using %s for %s due to %s\n", id, origType, r)
			}
		}
	}()
	id = obj.(interface{ AsGo() string }).AsGo()
	return
}

func coreType(e interface{}) string {
	return strings.Replace(fmt.Sprintf("%T", e), "core.", "", 1)
}

func coreTypeAsGo(e interface{}) string {
	s := strings.Replace(coreType(e), "*", "", 1)
	return strings.ToLower(s[0:1]) + s[1:]
}

func assertType(e interface{}) string {
	return ".(" + coreType(e) + ")"
}

func JoinStringFns(fns []func() string) string {
	strs := make([]string, len(fns))
	for ix, fn := range fns {
		strs[ix] = fn()
	}
	sort.Strings(strs)
	return strings.Join(strs, "")
}

func IsGoExprEmpty(s string) bool {
	return s == "" || (s[0:2] == "/*" && s[len(s)-2:] == "*/")
}

func maybeEmpty(s string, obj interface{}) string {
	if !IsGoExprEmpty(s) {
		return ""
	}
	return fmt.Sprintf("// (%T) ", obj)
}

func makeTypedTarget(target string, typedTarget bool, typeStr string) string {
	if typedTarget {
		return target
	}
	return target + typeStr
}

func metaHolder(target string, m Map, env *CodeEnv) string {
	res := noBang(emitMap(target+".meta", false, m, env))
	if IsGoExprEmpty(res) {
		return res
	}
	return fmt.Sprintf(`
	MetaHolder: MetaHolder{meta: %s},`[1:],
		res)
}

func MetaHolderField(target string, m MetaHolder, fields []string, env *CodeEnv) []string {
	f := metaHolder(target, m.meta, env)
	if IsGoExprEmpty(f) {
		return fields
	}
	return append(fields, f)
}

func infoHolder(target string, i InfoHolder, env *CodeEnv) string {
	res := noBang(i.info.Emit(target+".info", nil, env))
	if IsGoExprEmpty(res) {
		return res
	}
	if !notNil(res) {
		return ""
	}
	return fmt.Sprintf(`
	InfoHolder: InfoHolder{
	%s,
},`[1:],
		res)
}

func InfoHolderField(target string, m InfoHolder, fields []string, env *CodeEnv) []string {
	f := infoHolder(target, m, env)
	if IsGoExprEmpty(f) {
		return fields
	}
	return append(fields, f)
}

func emitString(target string, s string, env *CodeEnv) string {
	return NativeString{s}.Emit(target, nil, env)
}

func emitPtrToString(target string, s string, env *CodeEnv) string {
	return emitInternedString(target, s, env)
}

func emitInternedString(target string, s string, env *CodeEnv) (res string) {
	return "!&s_" + NameAsGo(s)
}

func directAssign(target string) string {
	cmp := strings.Split(target, ".")
	if len(cmp) < 2 {
		return target
	}
	final := cmp[len(cmp)-1]
	if final[0] == '(' && final[len(final)-1] == ')' {
		if len(cmp) > 2 {
			penultimate := cmp[len(cmp)-2]
			if penultimate[0] == '(' && penultimate[len(final)-1] == ')' {
				panic(fmt.Sprintf("directAssign(\"%s\")", target))
			}
		}
		return strings.Join(cmp[:len(cmp)-1], ".")
	}
	return target
}

func (b *Binding) Symbol() Symbol {
	return b.name
}

func (b *Binding) SymName() *string {
	return b.name.name
}

func (b *Binding) Index() int {
	return b.index
}

func (b *Binding) Frame() int {
	return b.frame
}

func (b *Binding) IsUsed() bool {
	return b.isUsed
}

func (b *Binding) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(b, actualPtr)
	env.Need[name] = b
	return fmt.Sprintf("&%s", name)
}

func (b *Binding) Finish(name string, env *CodeEnv) string {
	nameSet := noBang(b.name.Emit(name+".name", nil, env))
	if notNil(nameSet) {
		nameSet = fmt.Sprintf(`
	name: %s,
`[1:],
			nameSet)
	} else {
		nameSet = ""
	}

	static := fmt.Sprintf(`
var %s Binding = Binding{
%s	index: %d,
	frame: %d,
	isUsed: %v,
}
`[1:],
		name, nameSet, b.Index(), b.Frame(), b.IsUsed())

	return static
}

// func (env *CodeEnv) AddForm(o Object) {
// 	seq, ok := o.(Seq)
// 	if !ok {
// 		fmt.Printf("code.go: Skipping %s\n", o.ToString(false))
// 		return
// 	}
// 	first := seq.First()
// 	if v, ok := first.(Symbol); ok {
// 		switch v.ToString(false) {
// 		case "ns", "in-ns":
// 			fmt.Printf("core/code.go: Switching to namespace %s\n", o.ToString(false))
// 			seq = seq.Rest()
// 			if l, ok := seq.First().(*List); ok {
// 				if q, ok := l.First().(Symbol); !ok || *q.name != "quote" {
// 					fmt.Printf("code.go: unexpected form where namespace expected: %s\n", l.ToString(false))
// 					return
// 				}
// 				env.Namespace = GLOBAL_ENV.EnsureNamespace(l.Second().(Symbol))
// 			} else {
// 				env.Namespace = GLOBAL_ENV.EnsureNamespace(seq.First().(Symbol))
// 			}
// 			return
// 		}
// 	}
// 	/* Any other form, assume it'll affect the current
// 	/* namespace. The first time such a form is found within a
// 	/* namespace, capture its existing mappings so they are
// 	/* preserved with whatever values they have at runtime, while
// 	/* generating code to update the meta information for those
// 	/* mappings. */
// 	if env.Captured {
// 		return
// 	}
// 	for k, v := range env.Namespace.mappings {
// 		env.BaseMappings[k] = v
// 	}
// 	env.Captured = true
// }

func (env *CodeEnv) Emit() {
	statics := []string{}

	env.Runtime = append(env.Runtime, func() string {
		return fmt.Sprintf(`
	/* 00 */ _ns := GLOBAL_ENV.CurrentNamespace()
`[1:],
		)
	})

	for s, v := range env.Namespace.mappings {
		name := UniqueId(v, nil)
		symName := noBang(v.name.Emit("", nil, env))

		if _, ok := env.BaseMappings[s]; ok {
			env.Runtime = append(env.Runtime, func() string {
				return fmt.Sprintf(`
	/* 04 */ p_%s = _ns.UpdateVar(%s, %s)
`[1:],
					name, symName, name)
			})
		} else {
			env.Runtime = append(env.Runtime, func() string {
				return fmt.Sprintf(`
	/* 02 */ _ns.InternExistingVar(%s, &%s)
`[1:],
					symName, name)
			})
		}

		if _, ok := env.CodeWriterEnv.Generated[name]; ok {
			continue
		}

		env.CodeWriterEnv.Generated[name] = nil

		res := v.Emit("", nil, env)
		if res != "" && res[0] != '!' {
			panic(fmt.Sprintf("(Var)Emit() returned: %s", res))
		}

		env.CodeWriterEnv.Generated[name] = v
	}

	for {
		needLen := len(env.Need)
		for name, obj := range env.Need {
			if _, ok := env.Generated[name]; ok {
				continue
			}
			s := obj.Finish(name, env)
			env.Generated[name] = struct{}{}
			if s != "" {
				statics = append(statics, s)
			}
		}
		if len(env.Need) <= needLen {
			break
		}
		fmt.Printf("ANOTHER!! TIME!! was %d now %d\n", needLen, len(env.Need))
	}

	env.Statics += strings.Join(statics, "")
	env.Interns += JoinStringFns(env.Runtime)
}

func (p Position) Hash() uint32 {
	h := getHash()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(p.endLine))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, uint64(p.endColumn))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, uint64(p.startLine))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, uint64(p.startColumn))
	h.Write(b)
	if p.filename != nil {
		h.Write([]byte(*p.filename))
	}
	return h.Sum32()
}

func (p Position) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	fields := []string{}
	if p.endLine != 0 {
		fields = append(fields, fmt.Sprintf(`
	endLine: %d,`[1:],
			p.endLine))
	}
	if p.endColumn != 0 {
		fields = append(fields, fmt.Sprintf(`
	endColumn: %d,`[1:],
			p.endColumn))
	}
	if p.startLine != 0 {
		fields = append(fields, fmt.Sprintf(`
	startLine: %d,`[1:],
			p.startLine))
	}
	if p.startColumn != 0 {
		fields = append(fields, fmt.Sprintf(`
	startColumn: %d,`[1:],
			p.startColumn))
	}
	if p.filename != nil {
		imm, f := immediate(emitPtrToString(target+".filename", *p.filename, env))
		if imm && notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	filename: %s,`[1:],
				f))
		}
	}

	f := strings.Join(fields, "\n")
	if f != "" {
		f = "\n" + f + "\n"
	}
	return fmt.Sprintf(`Position{%s}`, f)
}

func (info *ObjectInfo) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	if info == nil {
		return "nil"
	}

	name := UniqueId(info, actualPtr)

	env.CodeWriterEnv.Need[name] = info

	return "!&" + name
}

func (obj *ObjectInfo) Finish(name string, env *CodeEnv) string {
	f := noBang(obj.Position.Emit(name+".Position", nil, env))
	if notNil(f) {
		f += ","
	}

	if !IsGoExprEmpty(f) {
		f = "\n" + f + "\n"
	}

	return fmt.Sprintf(`
var %s ObjectInfo = ObjectInfo{%s}
`,
		name, f)
}

func (sym Symbol) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(sym, nil)

	fields := []string{}
	fields = InfoHolderField(name, sym.InfoHolder, fields, env)
	fields = MetaHolderField(name, sym.MetaHolder, fields, env)

	f := ""
	strNs := ""
	imm := true
	if sym.ns != nil {
		var immNs bool
		immNs, strNs = immediate(emitInternedString("01", *sym.ns, env))
		f = strNs
		if !immNs {
			imm = false
		}
	}
	if notNil(f) {
		fields = append(fields, fmt.Sprintf(`
	ns: %s,`[1:],
			f))
	}

	strName := ""
	f = strName
	if sym.name != nil {
		var immName bool
		immName, strName = immediate(emitInternedString("01", *sym.name, env))
		f = strName
		if !immName {
			imm = false
		}
	}
	if notNil(f) {
		fields = append(fields, fmt.Sprintf(`
	name: %s,`[1:],
			f))
	}

	if sym.hash != 0 {
		fields = append(fields, fmt.Sprintf(`
	hash: %d,`[1:],
			sym.hash))
	}

	f = joinStatics(fields)

	if !imm {
		if target == "" {
			return fmt.Sprintf(`!Symbol{%s}`, f)
		}
		fn := func() string {
			return fmt.Sprintf(`
	/* 01 */ %s = Symbol{%s}
`[1:],
				directAssign(target), f)
		}
		env.Runtime = append(env.Runtime, fn)
		return "Symbol{}"
	}

	return fmt.Sprintf("!Symbol{%s}", f)
}

func (t *Type) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	if t == nil {
		return "nil"
	}
	name := noBang(emitInternedString("01", t.name, env))
	typeFn := func() string {
		return fmt.Sprintf(`
	/* 01 */ %s = TYPES[%s]
`[1:],
			directAssign(target), name)
	}
	env.Runtime = append(env.Runtime, typeFn)
	return "nil"
}

func emitProc(target string, p Proc, env *CodeEnv) string {
	return fmt.Sprintf("!Proc{fn: %s, name: %s}", p.name, strconv.Quote(p.name))
}

func (le *LocalEnv) Hash() uint32 {
	return HashPtr(uintptr(unsafe.Pointer(le)))
}

func (le *LocalEnv) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(le, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = le

		fields := []string{}
		f := deferObjectSeq(name+".bindings", &le.bindings, env)
		if f != "" {
			f = fmt.Sprintf("\t%sbindings: %s,", maybeEmpty(f, le.bindings), f)
		}
		fields = append(fields, f)
		if le.parent != nil {
			f := noBang(le.parent.Emit(name+".parent", nil, env))
			if f != "" {
				fields = append(fields, fmt.Sprintf("\t%sparent: %s,", maybeEmpty(f, le.parent), f))
			}
		}
		if le.frame != 0 {
			fields = append(fields, fmt.Sprintf("\tframe: %d,", le.frame))
		}
		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s LocalEnv = LocalEnv{%s}
`,
			name, f)
	}
	return "!&" + name
}

func emitFn(target string, fn *Fn, env *CodeEnv) string {
	name := UniqueId(fn, nil)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = fn
		fields := []string{}
		fields = InfoHolderField(name, fn.InfoHolder, fields, env)
		fields = MetaHolderField(name, fn.MetaHolder, fields, env)
		if fn.isMacro {
			fields = append(fields, "\tisMacro: true,")
		}
		if fn.fnExpr != nil {
			fnExpr := fn.fnExpr
			if len(fnExpr.arities) > 0 && fnExpr.arities[0].Position.startLine/10 == 73 {
				fmt.Printf("Fn@%p is %s\n", fn, name)
			}
			f := noBang(fnExpr.Emit(name+".fnExpr", nil, env))
			if f != "" {
				fields = append(fields, fmt.Sprintf("\t%sfnExpr: %s,", maybeEmpty(f, fnExpr), f))
			}
		}
		if fn.env != nil {
			f := noBang(fn.env.Emit(name+".env", nil, env))
			if f != "" {
				fields = append(fields, fmt.Sprintf("\t%senv: %s,", maybeEmpty(f, fn.env), f))
			}
		}
		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s Fn = Fn{%s%s}
`,
			name, metaHolder(name, fn.meta, env), f)
	}
	return "!&" + name
}

func (b Boolean) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	if b.B {
		return "!Boolean{B: true}"
	}
	return "!Boolean{B: false}"
}

func (m *MapSet) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(m, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = m
		fields := []string{}
		fields = InfoHolderField(name, m.InfoHolder, fields, env)
		fields = MetaHolderField(name, m.MetaHolder, fields, env)
		f := noBang(emitMap(name+".m", false, m.m, env))
		if f != "" {
			fields = append(fields, fmt.Sprintf(`
	m: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s MapSet = MapSet{%s}
`,
			name, f)
	}
	return "!&" + name
}

func emitMap(target string, typedTarget bool, m Map, env *CodeEnv) string {
	switch m := m.(type) {
	case *ArrayMap:
		return m.Emit(makeTypedTarget(target, typedTarget, ".(*ArrayMap)"), nil, env)
	case *HashMap:
		return m.Emit(makeTypedTarget(target, typedTarget, ".(*HashMap)"), nil, env)
	case nil:
		return ""
	}
	return fmt.Sprintf("nil /*ABEND: %T*/", m)
}

var spewConfig = &spew.ConfigState{
	Indent:       "",
	MaxDepth:     10,
	SortKeys:     true,
	SpewKeys:     true,
	NoDuplicates: true,
	UseOrdinals:  true,
}

func (l *List) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(l, actualPtr)
	status, ok := env.CodeWriterEnv.Generated[name]
	if !ok {
		env.CodeWriterEnv.Generated[name] = nil
		fields := []string{}

		fields = InfoHolderField(name, l.InfoHolder, fields, env)
		fields = MetaHolderField(name, l.MetaHolder, fields, env)
		f := noBang(emitObject(name+".first", false, &l.first, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	first: %s,`[1:],
				f))
		}
		if l.rest != nil {
			f := noBang(l.rest.Emit(name+".rest", nil, env))
			if notNil(f) {
				fields = append(fields, fmt.Sprintf(`
	rest: %s,`[1:],
					f))
			}
		}
		if l.count != 0 {
			fields = append(fields, fmt.Sprintf(`
	count: %d,`[1:],
				l.count))
		}
		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s List = List{%s}
`,
			name, f)
		env.CodeWriterEnv.Generated[name] = l
	} else if status == nil {
		fn := func() string {
			return fmt.Sprintf(`
	/* 03 */ %s = %s
`[1:],
				directAssign(target), "&"+name)
		}
		env.Runtime = append(env.Runtime, fn)
		return ""
	}
	return "!&" + name
}

func (v *Vector) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(v, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = v
		fields := []string{}
		fields = InfoHolderField(name, v.InfoHolder, fields, env)
		fields = MetaHolderField(name, v.MetaHolder, fields, env)
		fields = append(fields, fmt.Sprintf(`
	root: %s,`[1:],
			emitInterfaceSeq(name+".root", &v.root, env)))
		fields = append(fields, fmt.Sprintf(`
	tail: %s,`[1:],
			emitInterfaceSeq(name+".tail", &v.tail, env)))
		if v.count != 0 {
			fields = append(fields, fmt.Sprintf(`
	count: %d,`[1:],
				v.count))
		}
		if v.shift != 0 {
			fields = append(fields, fmt.Sprintf(`
	shift: %d,`[1:],
				v.shift))
		}
		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s Vector = Vector{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (m *ArrayMap) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(m, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = m
		fields := []string{}

		fields = InfoHolderField(name, m.InfoHolder, fields, env)
		fields = MetaHolderField(name, m.MetaHolder, fields, env)
		f := emitObjectSeq(name+".arr", &m.arr, env)
		if f != "" {
			fields = append(fields, fmt.Sprintf(`
	arr: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s ArrayMap = ArrayMap{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (m *HashMap) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(m, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = m
		fields := []string{}

		fields = InfoHolderField(name, m.InfoHolder, fields, env)
		fields = MetaHolderField(name, m.MetaHolder, fields, env)
		if m.count != 0 {
			fields = append(fields, fmt.Sprintf(`
	count: %d,`[1:],
				m.count))
		}
		f := noBang(emitInterface(name+".root", false, m.root, env))
		if f != "" {
			fields = append(fields, fmt.Sprintf(`
	root: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s HashMap = HashMap{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (b *BufferedReader) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(b, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = b
		fields := []string{}

		if b.hash != 0 {
			fields = append(fields, fmt.Sprintf(`
	hash: %d,`[1:],
				b.hash))

		}

		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s BufferedReader = BufferedReader{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (io *IOWriter) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	return "!(*IOWriter)(nil)"
}

func (ns *Namespace) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	if *ns.Name.name != "joker.core" {
		panic(fmt.Sprintf("code.go: (*Namespace)Emit() supports only ns=joker.core, not =%s\n", *ns.Name.name))
	}
	nsFn := func() string {
		return fmt.Sprintf(`
	/* 01 */ %s = _ns
`[1:],
			directAssign(target))
	}
	env.Runtime = append(env.Runtime, nsFn)
	return "nil"
}

func (s String) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(s, nil)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = s
		fields := []string{}
		fields = InfoHolderField(name, s.InfoHolder, fields, env)
		fields = append(fields, fmt.Sprintf(`
	S: %s,`[1:],
			strconv.Quote(s.S)))
		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s String = String{%s}
`,
			name, f)
	}
	return "!" + name
}

func (k Keyword) NsField() *string {
	return k.ns
}

func (k Keyword) NameField() *string {
	return k.name
}

func (k Keyword) HashField() uint32 {
	return k.hash
}

func (k Keyword) UniqueId() string {
	name := NameAsGo(*k.NameField())
	if k.NsField() != nil {
		return NameAsGo(*k.NsField()) + "_FW_" + name
	}
	return name
}

func (k Keyword) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := fmt.Sprintf("kw_%s", k.UniqueId())
	env.CodeWriterEnv.Need[name] = k

	fn := func(innerName string) func() string {
		return func() string {
			return fmt.Sprintf(`
	/* 01 */ %s = %s  // (Keyword)Emit()
`[1:],
				directAssign(target), innerName)
		}
	}(name)
	env.Runtime = append(env.Runtime, fn)

	return "nil"
}

func (k Keyword) Finish(name string, env *CodeEnv) string {
	immName, strName := immediate(emitInternedString(name+".name", *k.name, env))

	initName := ""
	if immName && notNil(strName) {
		initName = fmt.Sprintf(`
	name: %s,`[1:],
			strName)
	}

	immNs := true
	strNs := "nil"
	if k.NsField() != nil {
		ns := *k.ns
		immNs, strNs = immediate(emitInternedString(name+".ns", ns, env))
	}

	initNs := ""
	if immNs && notNil(strNs) {
		initNs = fmt.Sprintf(`
	ns: %s,`[1:],
			strNs)
	}

	fields := []string{}
	fields = InfoHolderField(name, k.InfoHolder, fields, env)
	if initNs != "" {
		fields = append(fields, initNs)
	}
	if initName != "" {
		fields = append(fields, initName)
	}
	if k.hash != 0 {
		fields = append(fields, fmt.Sprintf(`
	hash: %d,`[1:],
			k.hash))
	}

	f := strings.Join(fields, "\n")
	if !IsGoExprEmpty(f) {
		f = "\n" + f + "\n"
	}

	static := fmt.Sprintf(`
var %s Keyword = Keyword{%s}
`[1:],
		name, f)

	return static
}

func (i Int) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(i, nil)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = i
		fields := []string{}
		fields = InfoHolderField(name, i.InfoHolder, fields, env)
		fields = append(fields, fmt.Sprintf(`
	I: %d,`[1:],
			i.I))
		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s Int = Int{%s}
`,
			name, f)
	}
	return "!" + name
}

func (ch Char) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(ch, nil)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = ch
		fields := []string{}
		fields = InfoHolderField(name, ch.InfoHolder, fields, env)
		fields = append(fields, fmt.Sprintf(`
	Ch: '%c',`[1:],
			ch.Ch))
		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s Char = Char{%s}
`,
			name, f)
	}
	return "!" + name
}

func (d Double) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	dValue := strconv.FormatFloat(d.D, 'g', -1, 64)
	name := UniqueId(dValue, nil)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = d
		fields := []string{}
		fields = InfoHolderField(name, d.InfoHolder, fields, env)
		fields = append(fields, fmt.Sprintf(`
	D: %s,`[1:],
			dValue))
		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s Double = Double{%s}
`,
			name, f)
	}
	return "!" + name
}

func (n Nil) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(n, nil)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = n
		fields := []string{}
		fields = InfoHolderField(name, n.InfoHolder, fields, env)
		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s Nil = Nil{%s}
`,
			name, f)
	}
	return "!" + name
}

func emitInterface(target string, typedTarget bool, obj interface{}, env *CodeEnv) string {
	if obj == nil {
		return "nil"
	}
	switch obj := obj.(type) {
	case Symbol:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Symbol)"), nil, env)
	case *Var:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*Var)"), nil, env)
	case *Type:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*Type)"), nil, env)
	case Proc:
		return emitProc(makeTypedTarget(target, typedTarget, ".(Proc)"), obj, env)
	case *Fn:
		return emitFn(makeTypedTarget(target, typedTarget, ".(*Fn)"), obj, env)
	case Boolean:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Boolean)"), nil, env)
	case *MapSet:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*MapSet)"), nil, env)
	case *List:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*List)"), nil, env)
	case *Vector:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*Vector)"), nil, env)
	case *ArrayMap:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*ArrayMap)"), nil, env)
	case *HashMap:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*HashMap)"), nil, env)
	case *IOWriter:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*IOWriter)"), nil, env)
	case *Namespace:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*Namespace)"), nil, env)
	case *BufferedReader:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*BufferedReader)"), nil, env)
	case String:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(String)"), nil, env)
	case Keyword:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Keyword)"), nil, env)
	case Int:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Int)"), nil, env)
	case Char:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Char)"), nil, env)
	case Double:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Double)"), nil, env)
	case Nil:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Nil)"), nil, env)
	}
	return fmt.Sprintf("nil /*ABEND: unknown interface{} type %T: %+v*/", obj, obj)
}

func emitObject(target string, typedTarget bool, objPtr *Object, env *CodeEnv) string {
	if objPtr == nil {
		return ""
	}
	obj := *objPtr
	if obj == nil {
		return ""
	}
	switch obj := obj.(type) {
	case Symbol:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Symbol)"), nil, env)
	case *Var:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*Var)"), nil, env)
	case *Type:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*Type)"), nil, env)
	case Proc:
		return emitProc(makeTypedTarget(target, typedTarget, ".(Proc)"), obj, env)
	case *Fn:
		return emitFn(makeTypedTarget(target, typedTarget, ".(*Fn)"), obj, env)
	case Boolean:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Boolean)"), nil, env)
	case *MapSet:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*MapSet)"), nil, env)
	case *List:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*List)"), nil, env)
	case *Vector:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*Vector)"), nil, env)
	case *ArrayMap:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*ArrayMap)"), nil, env)
	case *HashMap:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*HashMap)"), nil, env)
	case *IOWriter:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*IOWriter)"), nil, env)
	case *Namespace:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*Namespace)"), nil, env)
	case *BufferedReader:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(*BufferedReader)"), nil, env)
	case String:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(String)"), nil, env)
	case Keyword:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Keyword)"), nil, env)
	case Int:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Int)"), nil, env)
	case Char:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Char)"), nil, env)
	case Double:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Double)"), nil, env)
	case Nil:
		return obj.Emit(makeTypedTarget(target, typedTarget, ".(Nil)"), nil, env)
	}
	return fmt.Sprintf("/*ABEND: unknown object type %T: %+v*/", obj, obj)
}

func (expr *LiteralExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = noBang(emitObject(name+".obj", false, &expr.obj, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	obj: %s,`[1:],
				f))
		}
		if expr.isSurrogate {
			fields = append(fields, `
	isSurrogate: true,`[1:])
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s LiteralExpr = LiteralExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func emitInterfaceSeq(target string, objects *[]interface{}, env *CodeEnv) string {
	objae := []string{}
	for ix, obj := range *objects {
		f := noBang(emitInterface(fmt.Sprintf("%s[%d]", target, ix), false, obj, env))
		objae = append(objae, fmt.Sprintf("\t%s%s,", maybeEmpty(f, obj), f))
	}
	ret := strings.Join(objae, "\n")
	if !IsGoExprEmpty(ret) {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]interface{}{%s}`, ret)
}

func emitSeq(target string, exprs *[]Expr, env *CodeEnv) string {
	exprae := []string{}
	for ix, _ := range *exprs {
		expr := &((*exprs)[ix])
		exprae = append(exprae, "\t"+noBang((*expr).Emit(fmt.Sprintf("%s[%d].(%s)", target, ix, coreType(expr)), expr, env))+",")
	}
	ret := strings.Join(exprae, "\n")
	if !IsGoExprEmpty(ret) {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]Expr{%s}`, ret)
}

func emitObjectSeq(target string, objs *[]Object, env *CodeEnv) string {
	objae := []string{}
	for ix, _ := range *objs {
		obj := &((*objs)[ix])
		objae = append(objae, "\t"+noBang(emitObject(fmt.Sprintf("%s[%d]", target, ix), false, obj, env))+",")
	}
	ret := strings.Join(objae, "\n")
	if !IsGoExprEmpty(ret) {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]Object{%s}`, ret)
}

func deferObjectSeq(target string, objs *[]Object, env *CodeEnv) string {
	objae := []string{}
	for ix, _ := range *objs {
		obj := &((*objs)[ix])
		objae = append(objae, "\tnil,")
		objFn := func(innerIx int) func() string {
			return func() string {
				el := fmt.Sprintf("%s[%d]", target, innerIx)
				return fmt.Sprintf(`
	/* 03 */ %s = %s  // deferObjectSeq[%d]
`[1:],
					directAssign(el), noBang(emitObject(el, false, obj, env)), innerIx)
			}
		}(ix) // Need an inner binding to capture the current val of ix
		env.Runtime = append(env.Runtime, objFn)
	}
	ret := strings.Join(objae, "\n")
	if !IsGoExprEmpty(ret) {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]Object{%s}`, ret)
}

func emitSymbolSeq(target string, syms *[]Symbol, env *CodeEnv) string {
	symv := []string{}
	for ix, _ := range *syms {
		sym := &((*syms)[ix])
		symv = append(symv, "\t"+noBang(sym.Emit(fmt.Sprintf("%s[%d]", target, ix), nil, env))+",")
	}
	ret := strings.Join(symv, "\n")
	if !IsGoExprEmpty(ret) {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]Symbol{%s}`, ret)
}

func emitFnArityExprSeq(target string, fns *[]FnArityExpr, env *CodeEnv) string {
	fnae := []string{}
	for ix, _ := range *fns {
		fn := &((*fns)[ix])
		if fn.Position.startLine/10 == 73 {
			fmt.Printf("FnArityExprSeq([%d]@%p %s)\n", ix, fn, fn.Position)
		}
		// "*" prefix for target indicates the value, not a reference to the value, is needed
		fnae = append(fnae, "\t"+indirect(noBang(fn.Emit(fmt.Sprintf("*%s[%d]", target, ix), nil, env)))+",")
	}
	ret := strings.Join(fnae, "\n")
	if !IsGoExprEmpty(ret) {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]FnArityExpr{%s}`, ret)
}

func emitCatchExprSeq(target string, ces []*CatchExpr, env *CodeEnv) string {
	ceae := []string{}
	for ix, ce := range ces {
		ceae = append(ceae, "\t"+noBang(ce.Emit(fmt.Sprintf("%s[%d]", target, ix), nil, env))+",")
	}
	ret := strings.Join(ceae, "\n")
	if !IsGoExprEmpty(ret) {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]*CatchExpr{%s}`, ret)
}

func (expr *VectorExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSeq(name+".v", &expr.v, env)
		fields = append(fields, fmt.Sprintf(`
	v: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s VectorExpr = VectorExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *SetExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSeq(name+".elements", &expr.elements, env)
		fields = append(fields, fmt.Sprintf(`
	elements: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s SetExpr = SetExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *MapExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSeq(name+".keys", &expr.keys, env)
		fields = append(fields, fmt.Sprintf(`
	keys: %s,`[1:],
			f))
		f = emitSeq(name+".values", &expr.values, env)
		fields = append(fields, fmt.Sprintf(`
	values: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s MapExpr = MapExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *IfExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = noBang(expr.cond.Emit(name+".cond"+assertType(expr.cond), nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	cond: %s,`[1:],
				f))
		}
		f = noBang(expr.positive.Emit(name+".positive"+assertType(expr.positive), nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	positive: %s,`[1:],
				f))
		}
		f = noBang(expr.negative.Emit(name+".negative"+assertType(expr.negative), nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	negative: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s IfExpr = IfExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

// func (expr *DefExpr) Emit(target string, env *CodeEnv) string {
// 	// p = append(p, DEF_EXPR)
// 	// p = expr.Pos().Emit(p, env)
// 	// p = expr.name.Emit(p, env)
// 	// p = emitExprOrNil(expr.value, p, env)
// 	// p = emitExprOrNil(expr.meta, p, env)
// 	// p = expr.vr.info.Emit(p, env)
// 	// return p
// 	if expr.value == nil {
// 		return "" // just (declare name), which can be ignored here
// 	}

// 	name := NameAsGo(*expr.name.name)

// 	vr := noBang(expr.vr.Emit(target+".vr", env))
// 	if vr != "" {
// 		vr = fmt.Sprintf(`
// 	vr: %s,
// `[1:],
// 			vr)

// 	}

// 	initial := fmt.Sprintf(`
// &DefExpr{
// 	Position: %s,
// %s	name: %s,
// 	value: %s,
// 	meta: %s,
// 	}
// `[1:],
// 		name,
// 		noBang(expr.Pos().Emit(target+".Position", env)),
// 		vr,
// 		noBang(expr.name.Emit(target+".name", env)),
// 		noBang(emitExprOrNil(target+".value"+assertType(expr.value), expr.value, env)),
// 		noBang(emitExprOrNil(target+".meta"+assertType(expr.meta), expr.meta, env)))

// 	return initial
// }

// func unpackDefExpr(p []byte, header *EmitHeader) (*DefExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	name, p := unpackSymbol(p, header)
// 	varName := name
// 	varName.ns = nil
// 	vr := header.GlobalEnv.CurrentNamespace().Intern(varName)
// 	value, p := UnpackExprOrNil(p, header)
// 	meta, p := UnpackExprOrNil(p, header)
// 	varInfo, p := unpackObjectInfo(p, header)
// 	updateVar(vr, varInfo, value, name)
// 	res := &DefExpr{
// 		Position: pos,
// 		vr:       vr,
// 		name:     name,
// 		value:    value,
// 		meta:     meta,
// 	}
// 	return res, p
// }

func (expr *CallExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = noBang(expr.callable.Emit(name+".callable"+assertType(expr.callable), nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	callable: %s,`[1:],
				f))
		}
		f = emitSeq(name+".args", &expr.args, env)
		fields = append(fields, fmt.Sprintf(`
	args: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s CallExpr = CallExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *RecurExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSeq(name+".args", &expr.args, env)
		fields = append(fields, fmt.Sprintf(`
	args: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s RecurExpr = RecurExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (vr *Var) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(vr, actualPtr)
	status, ok := env.CodeWriterEnv.Generated[vr]

	ptrToName := "&" + name
	_, isBase := env.BaseMappings[vr.name.name]
	if isBase {
		ptrToName = "p_" + name
	}

	if !ok {
		env.CodeWriterEnv.Generated[vr] = nil

		fields := []string{}
		fields = InfoHolderField(name, vr.InfoHolder, fields, env)
		fields = MetaHolderField(name, vr.MetaHolder, fields, env)
		if !isBase {
			f := noBang(vr.ns.Emit(name+".ns", nil, env))
			if notNil(f) {
				fields = append(fields, fmt.Sprintf(`
	ns: %s,`[1:],
					f))
			}
			f = noBang(vr.name.Emit(name+".name", nil, env))
			if notNil(f) {
				fields = append(fields, fmt.Sprintf(`
	name: %s,`[1:],
					f))
			}
		}
		f := noBang(emitObject(name+".Value", false, &vr.Value, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Value: %s,`[1:],
				f))
		}
		if vr.expr != nil {
			f = noBang(vr.expr.Emit(name+".expr", nil, env))
			if notNil(f) {
				fields = append(fields, fmt.Sprintf(`
	expr: %s,`[1:],
					f))
			}
		}
		if vr.isMacro {
			fields = append(fields, `
	isMacro: true,`[1:])
		}
		if vr.isPrivate {
			fields = append(fields, `
	isPrivate: true,`[1:])
		}
		if vr.isDynamic {
			fields = append(fields, `
	isDynamic: true,`[1:])
		}
		f = noBang(vr.taggedType.Emit(name+".taggedType", nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	taggedType: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s Var = Var{%s}
`,
			name, f)
		if isBase {
			env.Statics += fmt.Sprintf(`
var %s *Var
`,
				ptrToName)
		}
		env.CodeWriterEnv.Generated[vr] = vr
	}

	if isBase || (ok && status == nil) {
		if target == "" {
			return ""
		}
		fn := func() string {
			return fmt.Sprintf(`
	/* 05 */ %s = %s
`[1:],
				directAssign(target), ptrToName)
		}
		env.Runtime = append(env.Runtime, fn)
		return ""
	}

	return "!" + ptrToName
}

func (expr *VarRefExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if status, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = nil
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:], f))
		}
		f = noBang(expr.vr.Emit(name+".vr", nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	%svr: %s,`[1:], maybeEmpty(f, expr.vr), f))
		}
		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s VarRefExpr = VarRefExpr{%s}  // %p
`,
			name, f, expr)
		env.CodeWriterEnv.Generated[name] = expr
	} else if status == nil {
		fn := func() string {
			return fmt.Sprintf(`
	/* 01 */ %s = %s
`[1:],
				directAssign(target), "&"+name)
		}
		env.Runtime = append(env.Runtime, fn)
		return ""
	}

	return "!&" + name
}

// func (expr *SetMacroExpr) Emit(target string, env *CodeEnv) string {
// 	// p = append(p, SET_MACRO_EXPR)
// 	// p = expr.Pos().Emit(p, env)
// 	// p = expr.vr.Emit(p, env)
// 	// return p
// 	return "ABEND(*SetMacroExpr)"
// }

// func unpackSetMacroExpr(p []byte, header *EmitHeader) (*SetMacroExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	vr, p := unpackVar(p, header)
// 	res := &SetMacroExpr{
// 		Position: pos,
// 		vr:       vr,
// 	}
// 	return res, p
// }

func (expr *BindingExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:], f))
		}
		f = noBang(expr.binding.Emit(name+".binding", nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	%sbinding: %s,`[1:], maybeEmpty(f, expr.binding), f))
		}
		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s BindingExpr = BindingExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

// func (expr *MetaExpr) Emit(target string, env *CodeEnv) string {
// 	// p = append(p, META_EXPR)
// 	// p = expr.Pos().Emit(p, env)
// 	// p = expr.meta.Emit(p, env)
// 	// p = expr.expr.Emit(p, env)
// 	// return p
// 	return "ABEND(*MetaExpr)"
// }

func (expr *DoExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr

		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:], f))
		}
		f = emitSeq(name+".body", &expr.body, env)
		fields = append(fields, fmt.Sprintf(`
	%sbody: %s,`[1:], maybeEmpty(f, expr.body), f))
		if expr.isCreatedByMacro {
			fields = append(fields, fmt.Sprintf(`
	isCreatedByMacro: true,`))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s DoExpr = DoExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *FnArityExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		if target[0] == '*' {
			target = target[1:]
		} else {
			target = name
		}
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(target+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSymbolSeq(target+".args", &expr.args, env)
		fields = append(fields, fmt.Sprintf(`
	args: %s,`[1:],
			f))
		f = emitSeq(target+".body", &expr.body, env)
		fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
			f))
		f = noBang(expr.taggedType.Emit(target+".taggedType", nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	%staggedType: %s,`[1:],
				maybeEmpty(f, expr.taggedType), f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s FnArityExpr = FnArityExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *FnExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr

		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		if len(expr.arities) > 0 && expr.arities[0].Position.startLine/10 == 73 {
			fmt.Printf("FnExpr@%p is %s\n", expr, name)
		}
		f = emitFnArityExprSeq(name+".arities", &expr.arities, env)
		fields = append(fields, fmt.Sprintf(`
	arities: %s,`[1:],
			f))
		if expr.variadic != nil {
			f = noBang(expr.variadic.Emit(name+".variadic", nil, env))
			if notNil(f) {
				fields = append(fields, fmt.Sprintf(`
	variadic: %s,`[1:],
					f))
			}
		}
		f = noBang(expr.self.Emit(name+".self", nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	self: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s FnExpr = FnExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *LetExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSymbolSeq(name+".names", &expr.names, env)
		fields = append(fields, fmt.Sprintf(`
	names: %s,`[1:],
			f))
		f = emitSeq(name+".values", &expr.values, env)
		fields = append(fields, fmt.Sprintf(`
	values: %s,`[1:],
			f))
		f = emitSeq(name+".body", &expr.body, env)
		fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s LetExpr = LetExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *LoopExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSymbolSeq(name+".names", &expr.names, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	names: %s,`[1:],
				f))
		}
		f = emitSeq(name+".values", &expr.values, env)
		fields = append(fields, fmt.Sprintf(`
	values: %s,`[1:],
			f))
		f = emitSeq(name+".body", &expr.body, env)
		fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s LoopExpr = LoopExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *ThrowExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = noBang(expr.e.Emit(name+".e"+assertType(expr.e), nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	e: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s ThrowExpr = ThrowExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *CatchExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = noBang(expr.excType.Emit(name+".excType"+assertType(expr.excType), nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	excType: %s,`[1:],
				f))
		}
		f = noBang(expr.excSymbol.Emit(name+".excSymbol", nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	excSymbol: %s,`[1:],
				f))
		}
		f = emitSeq(name+".body", &expr.body, env)
		fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s CatchExpr = CatchExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *TryExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSeq(name+".body", &expr.body, env)
		fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
			f))
		f = emitCatchExprSeq(name+".catches", expr.catches, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	catches: %s,`[1:],
				f))
		}
		f = emitSeq(name+".finallyExpr", &expr.finallyExpr, env)
		fields = append(fields, fmt.Sprintf(`
	finallyExpr: %s,`[1:],
			f))

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s TryExpr = TryExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}
