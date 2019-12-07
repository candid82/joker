package core

import (
	"encoding/binary"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

type (
	CodeEnv struct {
		CodeWriterEnv *CodeWriterEnv
		Namespace     *Namespace
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
	return fmt.Sprintf(`
var %s = %s
`[1:],
		name, strconv.Quote(s.s))
}

func (s InternedString) Finish(name string, env *CodeEnv) string {
	if _, ok := env.CodeWriterEnv.BaseStrings[s.s]; ok {
		fn := func() string {
			return fmt.Sprintf(`
	/* 00 */ p_%s = STRINGS.Intern(%s)
`[1:],
				name, strconv.Quote(s.s))
		}
		env.Runtime = append(env.Runtime, fn)

		return fmt.Sprintf(`
var p_%s *string
`[1:],
			name)
	}

	fn := func() string {
		return fmt.Sprintf(`
	/* 00 */ STRINGS.InternExistingString(&%s)
`[1:],
			name)
	}
	env.Runtime = append(env.Runtime, fn)

	return fmt.Sprintf(`
var %s = %s
`[1:],
		name, strconv.Quote(s.s))
}

func NewCodeEnv(cwe *CodeWriterEnv) *CodeEnv {
	return &CodeEnv{
		CodeWriterEnv: cwe,
		Namespace:     GLOBAL_ENV.CoreNamespace,
		Need:          map[string]Finisher{},
		Generated:     map[interface{}]interface{}{},
	}
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

func uniqueName(target, prefix, f string, id, actual interface{}) string {
	if actual != nil {
		id = actual
	}
	return fmt.Sprintf("%s"+f, prefix, id)
}

func coreType(e interface{}) string {
	return strings.Replace(fmt.Sprintf("%T", e), "core.", "", 1)
}

func coreTypeAsGo(e interface{}) string {
	s := strings.Replace(coreType(e), "*", "PtrTo", 1)
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
	return fmt.Sprintf(`
	InfoHolder: InfoHolder{info: %s},`[1:],
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
	if _, ok := env.CodeWriterEnv.BaseStrings[s]; ok {
		name := "s_" + NameAsGo(s)
		env.CodeWriterEnv.Need[name] = InternedString{s}

		if target != "" {
			fn := func() string {
				return fmt.Sprintf(`
	/* 01 */ %s = p_%s
`[1:],
					target, name)
			}
			env.Runtime = append(env.Runtime, fn)
		}

		return "nil"
	}
	return "!" + ptrTo(noBang(emitString("", s, env)))
}

func emitInternedString(target string, s string, env *CodeEnv) string {
	internedStringVar := "s_" + NameAsGo(s)
	env.CodeWriterEnv.Need[internedStringVar] = InternedString{s}

	if _, ok := env.CodeWriterEnv.BaseStrings[s]; ok {
		return "!p_" + internedStringVar
	}

	if target != "" {
		fn := func() string {
			return fmt.Sprintf(`
	/* 00 */ %s = STRINGS.Intern(%s)
`[1:],
				target, strconv.Quote(s))
		}
		env.Runtime = append(env.Runtime, fn)
	}

	return "!" + ptrTo(internedStringVar)
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

func (b *Binding) UniqueId() string {
	isUsed := ""
	if b.IsUsed() {
		isUsed = "_used"
	}
	return fmt.Sprintf("%s_%d_%d%s", *b.SymName(), b.Index(), b.Frame(), isUsed)
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
	name := "b_" + NameAsGo(b.UniqueId())
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
var %s = Binding{
%s	index: %d,
	frame: %d,
	isUsed: %v,
}
`[1:],
		name, nameSet, b.Index(), b.Frame(), b.IsUsed())

	return static
}

func (env *CodeEnv) AddForm(o Object) {
	seq, ok := o.(Seq)
	if !ok {
		fmt.Printf("code.go: Skipping %s\n", o.ToString(false))
		return
	}
	first := seq.First()
	if v, ok := first.(Symbol); ok {
		switch v.ToString(false) {
		case "ns", "in-ns":
			fmt.Printf("core/code.go: Switching to namespace %s\n", o.ToString(false))
			seq = seq.Rest()
			if l, ok := seq.First().(*List); ok {
				if q, ok := l.First().(Symbol); !ok || *q.name != "quote" {
					fmt.Printf("code.go: unexpected form where namespace expected: %s\n", l.ToString(false))
					return
				}
				env.Namespace = GLOBAL_ENV.EnsureNamespace(l.Second().(Symbol))
			} else {
				env.Namespace = GLOBAL_ENV.EnsureNamespace(seq.First().(Symbol))
			}
		}
	}
}

func (env *CodeEnv) Emit() {
	statics := []string{}
	interns := []string{}

	interns = append(interns, fmt.Sprintf(`
	_ns := GLOBAL_ENV.CurrentNamespace()
`[1:],
	))

	for s, v := range env.Namespace.mappings {
		varName := NameAsGo(*s)
		name := "v_" + varName

		v_var := ""

		if v.Value != nil {
			v_value := indirect(emitObject(fmt.Sprintf("%s.Value.(%s)", name, coreType(v.Value)), true, &v.Value, env))
			if notNil(v_value) {
				intermediary := v_value[1:]
				if v_value[0] != '!' {
					intermediary = fmt.Sprintf("&value_%s", varName)
					statics = append(statics, fmt.Sprintf(`
var value_%s = %s
`[1:],
						varName, v_value))
				}
				v_var += fmt.Sprintf(`
	Value: %s,
`[1:],
					intermediary)
			}
		}

		if v.expr != nil {
			v_expr := indirect(v.expr.Emit("expr_"+name, nil, env))
			intermediary := v_expr[1:]
			if v_expr[0] != '!' {
				intermediary = fmt.Sprintf("&expr_%s", varName)
				statics = append(statics, fmt.Sprintf(`
var expr_%s = %s
`[1:],
					varName, v_expr))
			}
			v_var += fmt.Sprintf(`
	expr: %s,
`[1:],
				intermediary)
		}

		if v.isMacro {
			v_var += fmt.Sprintf(`
	isMacro: true,
`[1:])
		}

		if v.isPrivate {
			v_var += fmt.Sprintf(`
	isPrivate: true,
`[1:])
		}

		if v.isDynamic {
			v_var += fmt.Sprintf(`
	isDynamic: true,
`[1:])
		}

		if v.isUsed {
			v_var += fmt.Sprintf(`
	isUsed: true,
`[1:])
		}

		if v.isGloballyUsed {
			v_var += fmt.Sprintf(`
	isGloballyUsed: true,
`[1:])
		}

		v_tt := v.taggedType.Emit(name+".taggedType", nil, env)
		if notNil(v_tt) {
			intermediary := v_tt[1:]
			if v_tt[0] != '!' {
				intermediary = fmt.Sprintf("&taggedType_%s", varName)
				statics = append(statics, fmt.Sprintf(`
var taggedType_%s = %s
`[1:],
					v_tt))
			}
			v_var += fmt.Sprintf(`
	%staggedType: %s,
`[1:],
				maybeEmpty(v_tt, v.taggedType), intermediary)
		}

		if !IsGoExprEmpty(v_var) {
			v_var = `
` + v_var + `
`
		}
		info := infoHolder(name, v.InfoHolder, env)
		if info != "" {
			info = "\n" + info
		}
		meta := metaHolder(name, v.meta, env)
		if meta != "" {
			meta = "\n" + meta
		}
		v_var = fmt.Sprintf(`
var %s = Var{%s%s%s}
var p_%s = &%s
`[1:],
			name, info, meta, v_var, name, name)
		env.CodeWriterEnv.Generated[v] = v

		symName := noBang(v.name.Emit("", nil, env))

		interns = append(interns, fmt.Sprintf(`
	/* 00 */ _ns.InternExistingVar(%s, &%s)
`[1:],
			symName, name))

		statics = append(statics, v_var)
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
	env.Interns += strings.Join(interns, "") + JoinStringFns(env.Runtime)
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
		f := noBang(emitPtrToString(target+".filename", *p.filename, env))
		if notNil(f) {
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

	name := uniqueName(target, "objectInfo_", "%p", info, actualPtr)

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
var %s = ObjectInfo{%s}
`,
		name, f)
}

func (s Symbol) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	if s.name == nil {
		if s.ns == nil && s.hash == 0 {
			return ""
		}
		panic("Symbol{ABEND: No name!!}")
	}

	name := fmt.Sprintf("sym_%s", NameAsGo(*s.name))

	env.Need[name] = s

	if target == "" {
		return name
	}

	env.Runtime = append(env.Runtime, func() string {
		return fmt.Sprintf(`
	/* 02 */ %s = %s
`[1:],
			directAssign(target), name)
	})

	return "!Symbol{}"
}

func (sym Symbol) Finish(name string, env *CodeEnv) string {
	strName := emitInternedString(name+".name", *sym.name, env)

	strNs := "nil"
	if sym.ns != nil {
		ns := *sym.ns
		strNs = emitInternedString(name+".ns", ns, env)
	}

	fields := []string{}
	fields = InfoHolderField(name, sym.InfoHolder, fields, env)
	fields = MetaHolderField(name, sym.MetaHolder, fields, env)

	staticInitName, initName := immediate(strName)
	if staticInitName {
		initName = fmt.Sprintf(`
	name: %s,
`[1:],
			initName)
	} else {
		initName = ""
	}

	staticInitNs, initNs := immediate(strNs)
	if staticInitNs {
		initNs = fmt.Sprintf(`
	ns: %s,
`[1:],
			initNs)
	} else {
		initNs = ""
	}

	meta := strings.Join(fields, "\n")
	if !IsGoExprEmpty(meta) {
		meta = meta + "\n"
	}

	static := fmt.Sprintf(`
var %s = Symbol{
%s%s%s}
`[1:],
		name, meta, initNs, initName)

	if sym.hash != 0 {
		runtime := fmt.Sprintf(`
	/* 01 */ %s.hash = hashSymbol(%s, %s)
`[1:],
			name, ptrTo(strNs), ptrTo(strName))

		env.Runtime = append(env.Runtime, func(s string) func() string {
			return func() string { return s }
		}(runtime))
	}

	return static
}

func (t *Type) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	if t == nil {
		return "nil"
	}
	name := noBang(emitInternedString("", t.name, env))
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
	return "!" + p.name
}

func (le *LocalEnv) Hash() uint32 {
	return HashPtr(uintptr(unsafe.Pointer(le)))
}

func (le *LocalEnv) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "localEnv_", "%p", le, actualPtr)
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
var %s = LocalEnv{%s}
`,
			name, f)
	}
	return "!&" + name
}

func emitFn(target string, fn *Fn, env *CodeEnv) string {
	name := uniqueName(target, "fn_", "%p", fn, nil)
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
var %s = Fn{%s%s}
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
	name := uniqueName(target, "mapset_", "%p", m, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = m
		f := noBang(emitMap(name+".m", false, m.m, env))
		if f != "" {
			f = fmt.Sprintf("\t%sm: %s,", maybeEmpty(f, m.m), f)
		}
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = MapSet{%s}
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

func UniqueId(obj interface{}) string {
	h := getHash()
	h.Write(([]byte)(spewConfig.Sdump(obj)))
	return fmt.Sprintf("%s_%d", coreTypeAsGo(obj), h.Sum32())
}

func (l *List) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(l)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = nil
		fields := []string{}
		f := noBang(emitObject(name+".first", false, &l.first, env))
		if f != "" {
			fields = append(fields, fmt.Sprintf("\t%sfirst: %s,", maybeEmpty(f, l.first), f))
		}
		field := name + ".rest"
		if l.rest != nil {
			restName := UniqueId(l.rest)
			if status, found := env.CodeWriterEnv.Generated[restName]; found && status == nil {
				fieldFn := func() string {
					return fmt.Sprintf(`
	/* 03 */ %s = %s
`[1:],
						directAssign(field), noBang(l.rest.Emit(field, nil, env)))
				}
				env.Runtime = append(env.Runtime, fieldFn)
			} else {
				f := noBang(l.rest.Emit(field, nil, env))
				if f != "" {
					fields = append(fields, fmt.Sprintf("\t%srest: %s,", maybeEmpty(f, l.rest), f))
				}
			}
		}
		if l.count != 0 {
			fields = append(fields, fmt.Sprintf("\tcount: %d,", l.count))
		}
		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = List{%s}
`,
			name, f)
		env.CodeWriterEnv.Generated[name] = l
	}
	return "!&" + name
}

func (v *Vector) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := UniqueId(v)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = v
		fields := []string{}
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
var %s = Vector{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (m *ArrayMap) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "arrayMap_", "%p", m, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = m
		f := emitObjectSeq(name+".arr", &m.arr, env)
		if f != "" {
			f = fmt.Sprintf("\t%sarr: %s,", maybeEmpty(f, m.arr), f)
		}
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = ArrayMap{%s%s}
`,
			name, metaHolder(name, m.meta, env), f)
	}
	return "!&" + name
}

func (m *HashMap) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "hashMap_", "%p", m, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = m
		fields := []string{}
		if m.count != 0 {
			fields = append(fields, fmt.Sprintf("\tcount: %d,", m.count))
		}
		f := noBang(emitInterface(name+".root", false, &m.root, env))
		if f != "" {
			fields = append(fields, fmt.Sprintf("\t%sroot: %s,", maybeEmpty(f, m.root), f))
		}
		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = HashMap{%s%s}
`,
			name, metaHolder(name, m.meta, env), f)
	}
	return "!&" + name
}

func (b *BufferedReader) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "bufferedReader_", "%p", b, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = b
		fields := []string{}

		// if b != nil && b.Reader != nil && b.Reader.Fd() != os.Stdin {
		// 	panic(fmt.Sprintf("hey that is not right, it is %v", *b))
		// }

		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = BufferedReader{%s}
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
	/* 03 */ %s = _ns
`[1:], directAssign(target))
	}
	env.Runtime = append(env.Runtime, nsFn)
	return "nil"
}

func (s String) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "string_", "%d", s.Hash(), nil)
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
var %s = String{%s}
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
	strName := noBang(emitPtrToString(name+".name", *k.name, env))

	initName := ""
	if notNil(strName) {
		initName = fmt.Sprintf(`
	name: %s,
`[1:],
			strName)
	}

	strNs := "nil"
	if k.NsField() != nil {
		ns := *k.ns
		strNs = noBang(emitPtrToString(name+".ns", ns, env))
	}

	initNs := ""
	if notNil(strNs) {
		initNs = fmt.Sprintf(`
	ns: %s,
`[1:],
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

	f := strings.Join(fields, "\n")
	if !IsGoExprEmpty(f) {
		f = "\n" + f + "\n"
	}

	static := fmt.Sprintf(`
var %s = Keyword{%s}
`[1:],
		name, f)

	runtime := fmt.Sprintf(`
	/* 01 */ %s.hash = hashSymbol(%s, %s)
`[1:],
		name, ptrTo(strNs), ptrTo(strName))
	env.Runtime = append(env.Runtime, func(s string) func() string {
		return func() string { return s }
	}(runtime))

	return static
}

func (i Int) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "int_", "%d", i.I, nil)
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
var %s = Int{%s}
`,
			name, f)
	}
	return "!" + name
}

func (ch Char) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "char_", "%d", ch.Ch, nil)
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
var %s = Char{%s}
`,
			name, f)
	}
	return "!" + name
}

func (d Double) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	dValue := strconv.FormatFloat(d.D, 'g', -1, 64)
	name := uniqueName(target, "double_", "%s", NameAsGo(dValue), nil)
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
var %s = Double{%s}
`,
			name, f)
	}
	return "!" + name
}

func (n Nil) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	var hash uint32
	if n.InfoHolder.info != nil {
		hash = n.InfoHolder.info.Position.Hash()
	}
	name := uniqueName(target, "nil_", "%d", hash, nil)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = n
		fields := []string{}
		fields = InfoHolderField(name, n.InfoHolder, fields, env)
		f := strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = Nil{%s}
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
	obj := *objPtr
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
	name := uniqueName(target, "literalExpr_", "%p", expr, actualPtr)
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
var %s = LiteralExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func emitInterfaceSeq(target string, thingies *[]interface{}, env *CodeEnv) string {
	thingyae := []string{}
	for ix, _ := range *thingies {
		thingy := &((*thingies)[ix])
		f := noBang(emitInterface(fmt.Sprintf("%s[%d]", target, ix), false, thingy, env))
		thingyae = append(thingyae, fmt.Sprintf("\t%s%s,", maybeEmpty(f, thingy), f))
	}
	ret := strings.Join(thingyae, "\n")
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
		fnae = append(fnae, "\t"+indirect(noBang(fn.Emit(fmt.Sprintf("%s[%d]", target, ix), nil, env)))+",")
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
	name := uniqueName(target, "vectorExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	v: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = VectorExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *SetExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "setExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	elements: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = SetExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *MapExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "mapExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	keys: %s,`[1:],
				f))
		}
		f = emitSeq(name+".values", &expr.values, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	values: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = MapExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *IfExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "ifExpr_", "%p", expr, actualPtr)
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
var %s = IfExpr{%s}
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
	name := uniqueName(target, "callExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	args: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = CallExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *RecurExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "recurExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	args: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = RecurExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (vr *Var) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	// sym := *vr.name.name
	// g := NameAsGo(sym)
	// strName := noBang(emitInternedString(target+".name.name", sym, env))

	// 	runtimeDefineVarFn := func() string {
	// 		/* Defer this logic until interns are generated during EOF handling. */
	// 		if _, ok := env.CodeWriterEnv.Generated[vr]; ok {
	// 			return "\n"
	// 		}

	// 		env.CodeWriterEnv.Generated[vr] = vr

	// 		decl := fmt.Sprintf(`
	// var p_v_%s *Var
	// `[1:],
	// 			g)
	// 		env.Statics += decl

	// 		return fmt.Sprintf(`
	// 	/* 01 */ p_v_%s = GLOBAL_ENV.CoreNamespace.mappings[%s]
	// `,
	// 			g, strName)
	// 	}
	// 	env.Runtime = append(env.Runtime, runtimeDefineVarFn)

	// 	runtimeAssignFn := func() string {
	// 		return fmt.Sprintf(`
	// 	/* 02 */ %s = p_v_%s
	// `[1:],
	// 			directAssign(target), g)
	// 	}
	// 	env.Runtime = append(env.Runtime, runtimeAssignFn)

	return ""
}

func (expr *VarRefExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "varRefExpr_", "%p", expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[expr]; !ok {
		env.CodeWriterEnv.Generated[expr] = expr
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
var %s = VarRefExpr{%s}
`,
			name, f)
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
	name := uniqueName(target, "bindingExpr_", "%p", expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[expr]; !ok {
		env.CodeWriterEnv.Generated[expr] = expr
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
var %s = BindingExpr{%s}
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
	name := uniqueName(target, "doExpr_", "%p", expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[expr]; !ok {
		env.CodeWriterEnv.Generated[expr] = expr

		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:], f))
		}
		f = emitSeq(name+".body", &expr.body, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	%sbody: %s,`[1:], maybeEmpty(f, expr.body), f))
		}
		if expr.isCreatedByMacro {
			fields = append(fields, fmt.Sprintf(`
	isCreatedByMacro: true,`))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = DoExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *FnArityExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "fnArityExpr_", "%p", expr, actualPtr)
	if _, ok := env.CodeWriterEnv.Generated[name]; !ok {
		env.CodeWriterEnv.Generated[name] = expr
		fields := []string{}
		f := expr.Position.Emit(name+".Position", nil, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	Position: %s,`[1:],
				f))
		}
		f = emitSymbolSeq(name+".args", &expr.args, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	args: %s,`[1:],
				f))
		}
		f = emitSeq(name+".body", &expr.body, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
				f))
		}
		f = noBang(expr.taggedType.Emit(name+".taggedType", nil, env))
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
var %s = FnArityExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *FnExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "fnExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	arities: %s,`[1:],
				f))
		}
		if expr.variadic != nil {
			f = noBang(expr.variadic.Emit(name+".variadic", nil, env))
			if notNil(f) {
				fields = append(fields, fmt.Sprintf(`
	variadic: %s,
`[1:],
					f))
			}
		}
		f = noBang(expr.self.Emit(name+".self", nil, env))
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	self: %s,
`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = FnExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *LetExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "letExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	values: %s,`[1:],
				f))
		}
		f = emitSeq(name+".body", &expr.body, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = LetExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *LoopExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "loopExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	values: %s,`[1:],
				f))
		}
		f = emitSeq(name+".body", &expr.body, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = LoopExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *ThrowExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "throwExpr_", "%p", expr, actualPtr)
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
var %s = ThrowExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *CatchExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "catchExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = CatchExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}

func (expr *TryExpr) Emit(target string, actualPtr interface{}, env *CodeEnv) string {
	name := uniqueName(target, "tryExpr_", "%p", expr, actualPtr)
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
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	body: %s,`[1:],
				f))
		}
		f = emitCatchExprSeq(name+".catches", expr.catches, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	catches: %s,`[1:],
				f))
		}
		f = emitSeq(name+".finallyExpr", &expr.finallyExpr, env)
		if notNil(f) {
			fields = append(fields, fmt.Sprintf(`
	finallyExpr: %s,`[1:],
				f))
		}

		f = strings.Join(fields, "\n")
		if !IsGoExprEmpty(f) {
			f = "\n" + f + "\n"
		}
		env.Statics += fmt.Sprintf(`
var %s = TryExpr{%s}
`,
			name, f)
	}
	return "!&" + name
}
