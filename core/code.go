package core

import (
	"fmt"
	"strings"
)

type (
	CodeEnv struct {
		codeWriterEnv    *CodeWriterEnv
		Namespace        *Namespace
		Definitions      map[*string]struct{}
		Symbols          []*string
		Strings          map[*string]uint16
		Bindings         map[*Binding]int
		nextStringIndex  uint16
		nextBindingIndex int
		runtime          []func() string
		HaveVars         map[string]struct{}
	}

	CodeWriterEnv struct {
		NeedSyms map[*string]struct{}
		HaveSyms map[*string]struct{}
	}

	EmitHeader struct {
		GlobalEnv *Env
		Strings   []*string
		Bindings  []Binding
	}
)

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
}

func NameAsGo(name string) string {
	for _, t := range tr {
		name = strings.ReplaceAll(name, t[0], "_"+t[1]+"_")
	}
	return name
}

func noBang(s string) string {
	if s[0] == '!' {
		return s[1:]
	}
	return s
}

func indirect(s string) string {
	if s[0] == '&' {
		return s[1:]
	}
	return "*" + s
}

func (b *Binding) Emit(target string, p []byte, env *CodeEnv) string {
	// p = b.name.Emit(p, env)
	// p = appendInt(p, b.index)
	// p = appendInt(p, b.frame)
	// p = appendBool(p, b.isUsed)
	return "!(*Binding)(nil)"
}

// func unpackBinding(p []byte, header *EmitHeader) (Binding, []byte) {
// 	name, p := unpackSymbol(p, header)
// 	index, p := extractInt(p)
// 	frame, p := extractInt(p)
// 	isUsed, p := extractBool(p)
// 	return Binding{
// 		name:   name,
// 		index:  index,
// 		frame:  frame,
// 		isUsed: isUsed,
// 	}, p
// }

func NewCodeEnv(cwe *CodeWriterEnv) *CodeEnv {
	return &CodeEnv{
		codeWriterEnv: cwe,
		Namespace:     GLOBAL_ENV.CoreNamespace,
		Definitions:   make(map[*string]struct{}),
		Symbols:       []*string{},
		Strings:       make(map[*string]uint16),
		Bindings:      make(map[*Binding]int),
		HaveVars:      make(map[string]struct{}),
	}
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
		case "def", "defn", "defn-", "defmacro", "defonce", "defmulti", "defmethod":
			for {
				seq = seq.Rest()
				if seq == nil {
					break
				}
				next := seq.First()
				if sym, ok := next.(Symbol); ok && v.ns == nil && v.name != nil {
					if _, ok := env.Definitions[sym.name]; ok {
					} else {
						env.Symbols = append(env.Symbols, sym.name)
						env.Definitions[sym.name] = struct{}{}
					}
					return
				}
				fmt.Printf("code.go: strange symbol name in %s\n", v.ToString(false))
			}
		case "add-doc-and-meta":
			return // TODO: implement add-doc-and-meta
		case "doseq":
			return // TODO: implement doseq
		case "ns-unmap":
			return // TODO: implement ns-unmap
		case "ns", "in-ns":
			fmt.Printf("At %s\n", o.ToString(false))
			seq = seq.Rest()
			if l, ok := seq.First().(*List); ok {
				if q, ok := l.First().(Symbol); !ok || *q.name != "quote" {
					fmt.Printf("code.go: unexpected form where namespace expected: %s\n", l)
					return
				}
				env.Namespace = GLOBAL_ENV.EnsureNamespace(l.Second().(Symbol))
			} else {
				env.Namespace = GLOBAL_ENV.EnsureNamespace(seq.First().(Symbol))
			}
			return
		case "joker.core/refer", "comment", "set-macro__":
			return
		}
	}
	fmt.Printf("code.go: Ignoring %s\n", o.ToString(false))
}

func (env *CodeEnv) Emit() (string, string) {
	// var bp string
	// bp = appendInt(bp, len(env.Bindings))
	// for k, v := range env.Bindings {
	// 	bp = appendInt(bp, v)
	// 	bp = k.Emit(bp, env)
	// }
	// p = appendInt(p, len(env.Strings))
	// for k, v := range env.Strings {
	// 	p = appendUint16(p, v)
	// 	if k == nil {
	// 		p = appendInt(p, -1)
	// 	} else {
	// 		p = appendInt(p, len(*k))
	// 		p = append(p, *k...)
	// 	}
	// }
	// p = append(p, bp...)
	// return p
	code := ""
	interns := fmt.Sprintf(`
	_ns := GLOBAL_ENV.CurrentNamespace()
`[1:],
	)
	for ix, s := range env.Symbols {
		v, ok := env.Namespace.mappings[s]
		if !ok {
			fmt.Printf("code.go: cannot find %s [%d] in %s\n", *s, ix, *env.Namespace.Name.name)
			continue
		}

		name := NameAsGo(*s)

		code += fmt.Sprintf(`
var sym_%s = &Symbol{ns: nil}
`,
			name)
		env.codeWriterEnv.HaveSyms[s] = struct{}{}

		v_value := ""
		if v.Value != nil {
			v_value = emitObject("value_"+name+".Value", v.Value, env)
		}
		v_expr := ""
		if v.expr != nil {
			v_expr = v.expr.Emit("expr_"+name+".expr", env)
		}

		v_assign := ""
		if v_value != "" || v_expr != "" {
			v_assign = fmt.Sprintf("v_%s := ", name)
			env.HaveVars[name] = struct{}{}
		}

		interns += fmt.Sprintf(`
	string_%s := STRINGS.Intern("%s")
	sym_%s.name = string_%s
	%s_ns.Intern(*sym_%s)
`,
			name, *s, name, name, v_assign, name)

		if v_value != "" {
			intermediary := v_value[1:]
			if v_value[0] != '!' {
				intermediary = fmt.Sprintf("value_%s", name)
				code += fmt.Sprintf(`
var value_%s = %s
`[1:],
					name, v_value)
			}
			interns += fmt.Sprintf(`
	v_%s.Value = %s
`[1:],
				name, intermediary)
		}

		if v_expr != "" {
			intermediary := v_expr[1:]
			if v_expr[0] != '!' {
				intermediary = fmt.Sprintf("expr_%s", name)
				code += fmt.Sprintf(`
var expr_%s = %s
`[1:],
					name, v_expr)
			}
			interns += fmt.Sprintf(`
	v_%s.expr = %s
`[1:],
				name, intermediary)
		}
	}

	return code, interns + joinStringFns(env.runtime)
}

func joinStringFns(fns []func() string) string {
	strs := make([]string, len(fns))
	for ix, fn := range fns {
		strs[ix] = fn()
	}
	return strings.Join(strs, "")
}

// func UnpackHeader(p []byte, env *Env) (*EmitHeader, []byte) {
// 	stringCount, p := extractInt(p)
// 	strs := make([]*string, stringCount)
// 	for i := 0; i < stringCount; i++ {
// 		var index uint16
// 		var length int
// 		index, p = extractUInt16(p)
// 		length, p = extractInt(p)
// 		if length == -1 {
// 			strs[index] = nil
// 		} else {
// 			strs[index] = STRINGS.Intern(string(p[:length]))
// 			p = p[length:]
// 		}
// 	}
// 	header := &EmitHeader{
// 		GlobalEnv: env,
// 		Strings:   strs,
// 	}
// 	bindingCount, p := extractInt(p)
// 	bindings := make([]Binding, bindingCount)
// 	for i := 0; i < bindingCount; i++ {
// 		var index int
// 		var b Binding
// 		index, p = extractInt(p)
// 		b, p = unpackBinding(p, header)
// 		bindings[index] = b
// 	}
// 	header.Bindings = bindings
// 	return header, p
// }

func (env *CodeEnv) stringIndex(s *string) uint16 {
	index, ok := env.Strings[s]
	if ok {
		return index
	}
	env.Strings[s] = env.nextStringIndex
	env.nextStringIndex++
	return env.nextStringIndex - 1
}

func (env *CodeEnv) bindingIndex(b *Binding) int {
	index, ok := env.Bindings[b]
	if ok {
		return index
	}
	env.Bindings[b] = env.nextBindingIndex
	env.nextBindingIndex++
	return env.nextBindingIndex - 1
}

// func appendBool(p []byte, b bool) []byte {
// 	var bb byte
// 	if b {
// 		bb = 1
// 	}
// 	return append(p, bb)
// }

// func extractBool(p []byte) (bool, []byte) {
// 	var b bool
// 	if p[0] == 1 {
// 		b = true
// 	}
// 	return b, p[1:]
// }

// func appendUint16(p []byte, i uint16) []byte {
// 	pp := make([]byte, 2)
// 	binary.BigEndian.PutUint16(pp, i)
// 	p = append(p, pp...)
// 	return p
// }

// func extractUInt16(p []byte) (uint16, []byte) {
// 	return binary.BigEndian.Uint16(p[0:2]), p[2:]
// }

// func appendUint32(p []byte, i uint32) []byte {
// 	pp := make([]byte, 4)
// 	binary.BigEndian.PutUint32(pp, i)
// 	p = append(p, pp...)
// 	return p
// }

// func extractUInt32(p []byte) (uint32, []byte) {
// 	return binary.BigEndian.Uint32(p[0:4]), p[4:]
// }

// func appendInt(p []byte, i int) []byte {
// 	pp := make([]byte, 8)
// 	binary.BigEndian.PutUint64(pp, uint64(i))
// 	p = append(p, pp...)
// 	return p
// }

// func extractInt(p []byte) (int, []byte) {
// 	return int(binary.BigEndian.Uint64(p[0:8])), p[8:]
// }

func (pos Position) Emit(target string, env *CodeEnv) string {
	// p = appendInt(p, pos.startLine)
	// p = appendInt(p, pos.endLine)
	// p = appendInt(p, pos.startColumn)
	// p = appendInt(p, pos.endColumn)
	// p = appendUint16(p, env.stringIndex(pos.filename))
	// return p
	return "!(Position)(nil)"
}

// func unpackPosition(p []byte, header *EmitHeader) (pos Position, pp []byte) {
// 	pos.startLine, p = extractInt(p)
// 	pos.endLine, p = extractInt(p)
// 	pos.startColumn, p = extractInt(p)
// 	pos.endColumn, p = extractInt(p)
// 	i, p := extractUInt16(p)
// 	pos.filename = header.Strings[i]
// 	return pos, p
// }

func (info *ObjectInfo) Emit(target string, env *CodeEnv) string {
	// if info == nil {
	// 	return append(p, NULL)
	// }
	// p = append(p, NOT_NULL)
	// return info.Pos().Emit(p, env)
	return "!(*ObjectInfo)(nil)"
}

// func unpackObjectInfo(p []byte, header *EmitHeader) (*ObjectInfo, []byte) {
// 	if p[0] == NULL {
// 		return nil, p[1:]
// 	}
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	return &ObjectInfo{Position: pos}, p
// }

func EmitObjectOrNil(obj Object, env *CodeEnv) string {
	// if obj == nil {
	// 	return append(p, NULL)
	// }
	// p = append(p, NOT_NULL)
	// return packObject(obj, p, env)
	return "!(*Object)(nil)"
}

// func UnpackObjectOrNil(p []byte, header *EmitHeader) (Object, []byte) {
// 	if p[0] == NULL {
// 		return nil, p[1:]
// 	}
// 	return unpackObject(p[1:], header)
// }

func (s Symbol) Emit(target string, env *CodeEnv) string {
	if s.name == nil {
		return "Symbol{}"
	}
	env.codeWriterEnv.NeedSyms[s.name] = struct{}{}
	return fmt.Sprintf("*sym_%s", NameAsGo(*s.name))
}

// func unpackSymbol(p []byte, header *EmitHeader) (Symbol, []byte) {
// 	info, p := unpackObjectInfo(p, header)
// 	meta, p := UnpackObjectOrNil(p, header)
// 	iname, p := extractUInt16(p)
// 	ins, p := extractUInt16(p)
// 	hash, p := extractUInt32(p)
// 	res := Symbol{
// 		InfoHolder: InfoHolder{info: info},
// 		name:       header.Strings[iname],
// 		ns:         header.Strings[ins],
// 		hash:       hash,
// 	}
// 	if meta != nil {
// 		res.meta = meta.(Map)
// 	}
// 	return res, p
// }

func (t *Type) Emit(target string, env *CodeEnv) string {
	if t != nil {
		typeFn := func() string {
			return fmt.Sprintf(`
	%s = TYPES["%s"]
`,
				target, t.name)
		}
		env.runtime = append(env.runtime, typeFn)
	}
	return "nil"
}

// func unpackType(p []byte, header *EmitHeader) (*Type, []byte) {
// 	s, p := unpackSymbol(p, header)
// 	return TYPES[s.name], p
// }

func emitProc(target string, p Proc, env *CodeEnv) string {
	return "!" + p.name
}

func (le *LocalEnv) Emit(target string, env *CodeEnv) string {
	return "!(*LocalEnv)(nil)"
}

func emitFn(target string, fn *Fn, env *CodeEnv) string {
	fields := []string{}
	if fn.isMacro {
		fields = append(fields, "\tisMacro: true,")
	}
	if fn.fnExpr != nil {
		fields = append(fields, fmt.Sprintf("\tfnExpr: %s,", noBang(fn.fnExpr.Emit(target+".fnExpr", env))))
	}
	if fn.env != nil {
		fields = append(fields, fmt.Sprintf("\tenv: %s,", noBang(fn.env.Emit(target+".env", env))))
	}
	f := strings.Join(fields, "\n")
	if f != "" {
		f = "\n" + f + "\n"
	}
	return fmt.Sprintf("&Fn{%s}", f)
}

func (b Boolean) Emit(target string, env *CodeEnv) string {
	if b.B {
		return "&Boolean{B: true}"
	}
	return "&Boolean{B: false}"
}

func (m *MapSet) Emit(target string, env *CodeEnv) string {
	return "!(*MapSet)(nil)"
}

func (l *List) Emit(target string, env *CodeEnv) string {
	return "!(*List)(nil)"
}

func (v *Vector) Emit(target string, env *CodeEnv) string {
	return "!(*Vector)(nil)"
}

func (m *ArrayMap) Emit(target string, env *CodeEnv) string {
	return "!(*ArrayMap)(nil)"
}

func (m *HashMap) Emit(target string, env *CodeEnv) string {
	return "!(*HashMap)(nil)"
}

func (io *IOWriter) Emit(target string, env *CodeEnv) string {
	return "!(*IOWriter)(nil)"
}

func emitObject(target string, obj Object, env *CodeEnv) string {
	switch obj := obj.(type) {
	case Symbol:
		return obj.Emit(target, env)
	case *Var:
		return obj.Emit(target, env)
	case *Type:
		return obj.Emit(target, env)
	case Proc:
		return emitProc(target, obj, env)
	case *Fn:
		return emitFn(target, obj, env)
	case Boolean:
		return obj.Emit(target, env)
	case *MapSet:
		return obj.Emit(target, env)
	case *List:
		return obj.Emit(target, env)
	case *Vector:
		return obj.Emit(target, env)
	case *ArrayMap:
		return obj.Emit(target, env)
	case *HashMap:
		return obj.Emit(target, env)
	case Nil:
		return "Nil{}"
	case *IOWriter:
		return obj.Emit(target, env)
	default:
		return fmt.Sprintf("/*ABEND: unknown object type %T*/", obj)
	}
}

// func unpackObject(p []byte, header *EmitHeader) (Object, []byte) {
// 	switch p[0] {
// 	case SYMBOL_OBJ:
// 		return unpackSymbol(p[1:], header)
// 	case VAR_OBJ:
// 		return unpackVar(p[1:], header)
// 	case TYPE_OBJ:
// 		return unpackType(p[1:], header)
// 	case NULL:
// 		var size int
// 		size, p = extractInt(p[1:])
// 		obj := readFromReader(bytes.NewReader(p[:size]))
// 		return obj, p[size:]
// 	default:
// 		panic(RT.NewError(fmt.Sprintf("Unknown object tag: %d", p[0])))
// 	}
// }

func (expr *LiteralExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, LITERAL_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = appendBool(p, expr.isSurrogate)
	// p = packObject(expr.obj, p, env)
	// return p
	return "!(*LiteralExpr)(nil)"
}

// func unpackLiteralExpr(p []byte, header *EmitHeader) (*LiteralExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	isSurrogate, p := extractBool(p)
// 	obj, p := unpackObject(p, header)
// 	res := &LiteralExpr{
// 		obj:         obj,
// 		Position:    pos,
// 		isSurrogate: isSurrogate,
// 	}
// 	return res, p
// }

func emitSeq(target string, exprs []Expr, env *CodeEnv) string {
	exprae := []string{}
	for ix, expr := range exprs {
		exprae = append(exprae, "\t"+noBang(expr.Emit(fmt.Sprintf("%s[%d]", target, ix), env))+",")
	}
	ret := strings.Join(exprae, "\n")
	if ret != "" {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]Expr{%s}`, ret)
}

// func unpackSeq(p []byte, header *EmitHeader) ([]Expr, []byte) {
// 	c, p := extractInt(p)
// 	res := make([]Expr, c)
// 	for i := 0; i < c; i++ {
// 		res[i], p = UnpackExpr(p, header)
// 	}
// 	return res, p
// }

func emitSymbolSeq(target string, syms []Symbol, env *CodeEnv) string {
	symv := []string{}
	for ix, sym := range syms {
		symv = append(symv, "\t"+noBang(sym.Emit(fmt.Sprintf("%s[%d]", target, ix), env))+",")
	}
	ret := strings.Join(symv, "\n")
	if ret != "" {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]Symbol{%s}`, ret)
}

// func unpackSymbolSeq(p []byte, header *EmitHeader) ([]Symbol, []byte) {
// 	c, p := extractInt(p)
// 	res := make([]Symbol, c)
// 	for i := 0; i < c; i++ {
// 		res[i], p = unpackSymbol(p, header)
// 	}
// 	return res, p
// }

func emitFnArityExprSeq(target string, fns []FnArityExpr, env *CodeEnv) string {
	fnae := []string{}
	for ix, fn := range fns {
		fnae = append(fnae, "\t"+indirect(noBang(fn.Emit(fmt.Sprintf("%s[%d]", target, ix), env)))+",")
	}
	ret := strings.Join(fnae, "\n")
	if ret != "" {
		ret = "\n" + ret + "\n"
	}
	return fmt.Sprintf(`[]FnArityExpr{%s}`, ret)
}

func emitCatchExprSeq(s []*CatchExpr, env *CodeEnv) string {
	// p = appendInt(p, len(s))
	// for _, e := range s {
	// 	p = e.Emit(p, env)
	// }
	// return p
	return "!(*CatchExprSeq)(nil)"
}

// func unpackCatchExprSeq(p []byte, header *EmitHeader) ([]*CatchExpr, []byte) {
// 	c, p := extractInt(p)
// 	res := make([]*CatchExpr, c)
// 	for i := 0; i < c; i++ {
// 		res[i], p = unpackCatchExpr(p, header)
// 	}
// 	return res, p
// }

func (expr *VectorExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, VECTOR_EXPR)
	// p = expr.Pos().Emit(p, env)
	// return packSeq(p, expr.v, env)
	return "!(*VectorExpr)(nil)"
}

// func unpackVectorExpr(p []byte, header *EmitHeader) (*VectorExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	v, p := unpackSeq(p, header)
// 	res := &VectorExpr{
// 		Position: pos,
// 		v:        v,
// 	}
// 	return res, p
// }

func (expr *SetExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, SET_EXPR)
	// p = expr.Pos().Emit(p, env)
	// return packSeq(p, expr.elements, env)
	return "!(*SetExpr)(nil)"
}

// func unpackSetExpr(p []byte, header *EmitHeader) (*SetExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	v, p := unpackSeq(p, header)
// 	res := &SetExpr{
// 		Position: pos,
// 		elements: v,
// 	}
// 	return res, p
// }

func (expr *MapExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, MAP_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSeq(p, expr.keys, env)
	// p = packSeq(p, expr.values, env)
	// return p
	return "!(*MapExpr)(nil)"
}

// func unpackMapExpr(p []byte, header *EmitHeader) (*MapExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	ks, p := unpackSeq(p, header)
// 	vs, p := unpackSeq(p, header)
// 	res := &MapExpr{
// 		Position: pos,
// 		keys:     ks,
// 		values:   vs,
// 	}
// 	return res, p
// }

func (expr *IfExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, IF_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.cond.Emit(p, env)
	// p = expr.positive.Emit(p, env)
	// p = expr.negative.Emit(p, env)
	// return p
	return "!(*IfExpr)(nil)"
}

// func unpackIfExpr(p []byte, header *EmitHeader) (*IfExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	cond, p := UnpackExpr(p, header)
// 	positive, p := UnpackExpr(p, header)
// 	negative, p := UnpackExpr(p, header)
// 	res := &IfExpr{
// 		Position: pos,
// 		positive: positive,
// 		negative: negative,
// 		cond:     cond,
// 	}
// 	return res, p
// }

func (expr *DefExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, DEF_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.name.Emit(p, env)
	// p = emitExprOrNil(expr.value, p, env)
	// p = emitExprOrNil(expr.meta, p, env)
	// p = expr.vr.info.Emit(p, env)
	// return p
	if expr.value == nil {
		return "" // just (declare name), which can be ignored here
	}

	name := NameAsGo(*expr.name.name)

	initial := fmt.Sprintf(`
&DefExpr{
	Position: %s,
	vr: %s,
	name: %s,
	value: %s,
	meta: %s,
	}
`[1:],
		name,
		noBang(expr.Pos().Emit(target+".Position", env)),
		noBang(expr.vr.Emit(target+".vr", env)),
		noBang(expr.name.Emit(target+".name", env)),
		noBang(emitExprOrNil(target+".value", expr.value, env)),
		noBang(emitExprOrNil(target+".meta", expr.meta, env)))

	env.codeWriterEnv.HaveSyms[expr.name.name] = struct{}{}
	return fmt.Sprintf(`
var sym_%s = &Symbol{}
%s`,
		name, initial)
}

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

func (expr *CallExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, CALL_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.callable.Emit(p, env)
	// p = packSeq(p, expr.args, env)
	// return p
	return fmt.Sprintf(`&CallExpr{
	callable: %s,
	args: %s,
}`,
		noBang(expr.callable.Emit(target+".callable", env)),
		emitSeq(target+".args", expr.args, env))
}

// func unpackCallExpr(p []byte, header *EmitHeader) (*CallExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	callable, p := UnpackExpr(p, header)
// 	args, p := unpackSeq(p, header)
// 	res := &CallExpr{
// 		Position: pos,
// 		callable: callable,
// 		args:     args,
// 	}
// 	return res, p
// }

func (expr *RecurExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, RECUR_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSeq(p, expr.args, env)
	// return p
	return "!(*RecurExpr)(nil)"
}

// func unpackRecurExpr(p []byte, header *EmitHeader) (*RecurExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	args, p := unpackSeq(p, header)
// 	res := &RecurExpr{
// 		Position: pos,
// 		args:     args,
// 	}
// 	return res, p
// }

func (vr *Var) Emit(target string, env *CodeEnv) string {
	// p = vr.ns.Name.Emit(p, env)
	// p = vr.name.Emit(p, env)
	// return p
	ns := *vr.ns.Name.name
	sym := *vr.name.name
	g := NameAsGo(sym)

	runtimeDefineVarFn := func() string {
		/* Defer this logic until interns are generated during EOF handling. */
		if _, ok := env.HaveVars[g]; ok {
			return ""
		}
		env.HaveVars[g] = struct{}{}
		return fmt.Sprintf(`
	v_%s := GLOBAL_ENV.FindNamespace("%s").mappings["%s"]
	if v_%s == nil {
 		panic(RT.NewError("Error unpacking var: cannot find var %s/%s"))
 	}
`,
			g, ns, sym, g, ns, sym)
	}
	env.runtime = append(env.runtime, runtimeDefineVarFn)

	runtimeAssignFn := func() string {
		return fmt.Sprintf(`
	%s = v_%s
`[1:],
			target, g)
	}
	env.runtime = append(env.runtime, runtimeAssignFn)

	return "!(*Var)(nil)" // TODO: Runtime initialization needed!
}

// func unpackVar(p []byte, header *EmitHeader) (*Var, []byte) {
// 	nsName, p := unpackSymbol(p, header)
// 	name, p := unpackSymbol(p, header)
// 	vr := GLOBAL_ENV.FindNamespace(nsName).mappings[name.name]
// 	if vr == nil {
// 		panic(RT.NewError("Error unpacking var: cannot find var " + *nsName.name + "/" + *name.name))
// 	}
// 	return vr, p
// }

func (expr *VarRefExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, VARREF_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.vr.Emit(p, env)
	// return p
	return fmt.Sprintf(`&VarRefExpr{
	vr: %s,
}`,
		noBang(expr.vr.Emit(target+".vr", env)))
}

// func unpackVarRefExpr(p []byte, header *EmitHeader) (*VarRefExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	vr, p := unpackVar(p, header)
// 	res := &VarRefExpr{
// 		Position: pos,
// 		vr:       vr,
// 	}
// 	return res, p
// }

func (expr *SetMacroExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, SET_MACRO_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.vr.Emit(p, env)
	// return p
	return "!(*SetMacroExpr)(nil)"
}

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

func (expr *BindingExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, BINDING_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = appendInt(p, env.bindingIndex(expr.binding))
	// return p
	return "!(*BindingExpr)(nil)"
}

// func unpackBindingExpr(p []byte, header *EmitHeader) (*BindingExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	index, p := extractInt(p)
// 	res := &BindingExpr{
// 		Position: pos,
// 		binding:  &header.Bindings[index],
// 	}
// 	return res, p
// }

func (expr *MetaExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, META_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.meta.Emit(p, env)
	// p = expr.expr.Emit(p, env)
	// return p
	return "!(*MetaExpr)(nil)"
}

// func unpackMetaExpr(p []byte, header *EmitHeader) (*MetaExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	meta, p := unpackMapExpr(p, header)
// 	expr, p := UnpackExpr(p, header)
// 	res := &MetaExpr{
// 		Position: pos,
// 		meta:     meta,
// 		expr:     expr,
// 	}
// 	return res, p
// }

func (expr *DoExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, DO_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSeq(p, expr.body, env)
	// return p
	return "!(*DoExpr)(nil)"
}

// func unpackDoExpr(p []byte, header *EmitHeader) (*DoExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	body, p := unpackSeq(p, header)
// 	res := &DoExpr{
// 		Position: pos,
// 		body:     body,
// 	}
// 	return res, p
// }

func (expr *FnArityExpr) Emit(target string, env *CodeEnv) string {
	if expr == nil {
		return "(*FnArityExpr)(nil)"
	}

	return fmt.Sprintf(`&FnArityExpr{
	args: %s,
	body: %s,
	taggedType: %s,
}`,
		emitSymbolSeq(target+".args", expr.args, env),
		emitSeq(target+".body", expr.body, env),
		noBang(expr.taggedType.Emit(target+".taggedType", env)))
}

// func (expr *FnArityExpr) Emit(env *CodeEnv) string {
// 	// p = append(p, FN_ARITY_EXPR)
// 	// p = expr.Pos().Emit(p, env)
// 	// p = packSymbolSeq(p, expr.args, env)
// 	// p = packSeq(p, expr.body, env)
// 	// if expr.taggedType != nil {
// 	// 	p = append(p, NOT_NULL)
// 	// 	p = appendUint16(p, env.stringIndex(STRINGS.Intern(expr.taggedType.name)))
// 	// } else {
// 	// 	p = append(p, NULL)
// 	// }
// 	// return p
// 	return "!(*FnArityExpr)(nil)"
// }

// func unpackFnArityExpr(p []byte, header *EmitHeader) (*FnArityExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	args, p := unpackSymbolSeq(p, header)
// 	body, p := unpackSeq(p, header)
// 	var taggedType *Type
// 	if p[0] == NULL {
// 		p = p[1:]
// 	} else {
// 		p = p[1:]
// 		var i uint16
// 		i, p = extractUInt16(p)
// 		taggedType = TYPES[header.Strings[i]]
// 	}
// 	res := &FnArityExpr{
// 		Position:   pos,
// 		body:       body,
// 		args:       args,
// 		taggedType: taggedType,
// 	}
// 	return res, p
// }

func (expr *FnExpr) Emit(target string, env *CodeEnv) string {
	return fmt.Sprintf(`&FnExpr{
	arities: %s,
	variadic: %s,
	self: %s,
}`,
		emitFnArityExprSeq(target+".arities", expr.arities, env),
		noBang(emitExprOrNil(target+".variadic", expr.variadic, env)),
		noBang(expr.self.Emit(target+".self", env)))
}

// func unpackFnExpr(p []byte, header *EmitHeader) (*FnExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	arities, p := unpackFnArityExprSeq(p, header)
// 	var variadic *FnArityExpr
// 	if p[0] == NULL {
// 		p = p[1:]
// 	} else {
// 		p = p[1:]
// 		variadic, p = unpackFnArityExpr(p, header)
// 	}
// 	self, p := unpackSymbol(p, header)
// 	res := &FnExpr{
// 		Position: pos,
// 		arities:  arities,
// 		variadic: variadic,
// 		self:     self,
// 	}
// 	return res, p
// }

func (expr *LetExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, LET_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSymbolSeq(p, expr.names, env)
	// p = packSeq(p, expr.values, env)
	// p = packSeq(p, expr.body, env)
	// return p
	return "!(*LetExpr)(nil)"
}

// func unpackLetExpr(p []byte, header *EmitHeader) (*LetExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	names, p := unpackSymbolSeq(p, header)
// 	values, p := unpackSeq(p, header)
// 	body, p := unpackSeq(p, header)
// 	res := &LetExpr{
// 		Position: pos,
// 		names:    names,
// 		values:   values,
// 		body:     body,
// 	}
// 	return res, p
// }

func (expr *LoopExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, LOOP_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSymbolSeq(p, expr.names, env)
	// p = packSeq(p, expr.values, env)
	// p = packSeq(p, expr.body, env)
	// return p
	return "!(*LoopExpr)(nil)"
}

// func unpackLoopExpr(p []byte, header *EmitHeader) (*LoopExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	names, p := unpackSymbolSeq(p, header)
// 	values, p := unpackSeq(p, header)
// 	body, p := unpackSeq(p, header)
// 	res := &LoopExpr{
// 		Position: pos,
// 		names:    names,
// 		values:   values,
// 		body:     body,
// 	}
// 	return res, p
// }

func (expr *ThrowExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, THROW_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.e.Emit(p, env)
	// return p
	return "!(*ThrowExpr)(nil)"
}

// func unpackThrowExpr(p []byte, header *EmitHeader) (*ThrowExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	e, p := UnpackExpr(p, header)
// 	res := &ThrowExpr{
// 		Position: pos,
// 		e:        e,
// 	}
// 	return res, p
// }

func (expr *CatchExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, CATCH_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = appendUint16(p, env.stringIndex(STRINGS.Intern(expr.excType.name)))
	// p = expr.excSymbol.Emit(p, env)
	// p = packSeq(p, expr.body, env)
	// return p
	return "!(*CatchExpr)(nil)"
}

// func unpackCatchExpr(p []byte, header *EmitHeader) (*CatchExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	i, p := extractUInt16(p)
// 	typeName := header.Strings[i]
// 	excSymbol, p := unpackSymbol(p, header)
// 	body, p := unpackSeq(p, header)
// 	res := &CatchExpr{
// 		Position:  pos,
// 		excSymbol: excSymbol,
// 		body:      body,
// 		excType:   TYPES[typeName],
// 	}
// 	return res, p
// }

func (expr *TryExpr) Emit(target string, env *CodeEnv) string {
	// p = append(p, TRY_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSeq(p, expr.body, env)
	// p = packCatchExprSeq(p, expr.catches, env)
	// p = packSeq(p, expr.finallyExpr, env)
	// return p
	return "!(*TryExpr)(nil)"
}

// func unpackTryExpr(p []byte, header *EmitHeader) (*TryExpr, []byte) {
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	body, p := unpackSeq(p, header)
// 	catches, p := unpackCatchExprSeq(p, header)
// 	finallyExpr, p := unpackSeq(p, header)
// 	res := &TryExpr{
// 		Position:    pos,
// 		body:        body,
// 		catches:     catches,
// 		finallyExpr: finallyExpr,
// 	}
// 	return res, p
// }

func emitExprOrNil(target string, expr Expr, env *CodeEnv) string {
	if expr == nil {
		return "nil"
	}
	return expr.Emit(target, env)
}

// func UnpackExprOrNil(p []byte, header *EmitHeader) (Expr, []byte) {
// 	if p[0] == NULL {
// 		return nil, p[1:]
// 	}
// 	return UnpackExpr(p[1:], header)
// }

// func UnpackExpr(p []byte, header *EmitHeader) (Expr, []byte) {
// 	switch p[0] {
// 	case LITERAL_EXPR:
// 		return unpackLiteralExpr(p, header)
// 	case VECTOR_EXPR:
// 		return unpackVectorExpr(p, header)
// 	case MAP_EXPR:
// 		return unpackMapExpr(p, header)
// 	case SET_EXPR:
// 		return unpackSetExpr(p, header)
// 	case IF_EXPR:
// 		return unpackIfExpr(p, header)
// 	case DEF_EXPR:
// 		return unpackDefExpr(p, header)
// 	case CALL_EXPR:
// 		return unpackCallExpr(p, header)
// 	case RECUR_EXPR:
// 		return unpackRecurExpr(p, header)
// 	case META_EXPR:
// 		return unpackMetaExpr(p, header)
// 	case DO_EXPR:
// 		return unpackDoExpr(p, header)
// 	case FN_ARITY_EXPR:
// 		return unpackFnArityExpr(p, header)
// 	case FN_EXPR:
// 		return unpackFnExpr(p, header)
// 	case LET_EXPR:
// 		return unpackLetExpr(p, header)
// 	case LOOP_EXPR:
// 		return unpackLoopExpr(p, header)
// 	case THROW_EXPR:
// 		return unpackThrowExpr(p, header)
// 	case CATCH_EXPR:
// 		return unpackCatchExpr(p, header)
// 	case TRY_EXPR:
// 		return unpackTryExpr(p, header)
// 	case VARREF_EXPR:
// 		return unpackVarRefExpr(p, header)
// 	case SET_MACRO_EXPR:
// 		return unpackSetMacroExpr(p, header)
// 	case BINDING_EXPR:
// 		return unpackBindingExpr(p, header)
// 	default:
// 		panic(RT.NewError(fmt.Sprintf("Unknown pack tag: %d", p[0])))
// 	}
// }
