package core

import (
	"fmt"
	"strings"
)

type (
	CodeEnv struct {
		Namespace        *Namespace
		Definitions      map[*string]struct{}
		Symbols          []*string
		Strings          map[*string]uint16
		Bindings         map[*Binding]int
		nextStringIndex  uint16
		nextBindingIndex int
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
}

func nameAsGo(name string) string {
	for _, t := range tr {
		name = strings.ReplaceAll(name, t[0], "_"+t[1]+"_")
	}
	return name
}

func (b *Binding) Emit(p []byte, env *CodeEnv) string {
	// p = b.name.Emit(p, env)
	// p = appendInt(p, b.index)
	// p = appendInt(p, b.frame)
	// p = appendBool(p, b.isUsed)
	return "nil /*Binding*/"
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

func NewCodeEnv() *CodeEnv {
	return &CodeEnv{
		Namespace:   GLOBAL_ENV.CoreNamespace,
		Definitions: make(map[*string]struct{}),
		Symbols:     []*string{},
		Strings:     make(map[*string]uint16),
		Bindings:    make(map[*Binding]int),
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
	interns := ""
	for ix, s := range env.Symbols {
		v, ok := env.Namespace.mappings[s]
		if !ok {
			fmt.Printf("code.go: cannot find %s [%d] in %s\n", *s, ix, *env.Namespace.Name.name)
			continue
		}

		name := nameAsGo(*s)

		code += fmt.Sprintf(`
var sym_%s = &Symbol{ns: nil}
`,
			name)

		v_value := ""
		if v.Value != nil {
			v_value = emitObject(v.Value, env)
		}
		v_expr := ""
		if v.expr != nil {
			v_expr = v.expr.Emit(env)
		}

		interns += fmt.Sprintf(`
	string_%s := STRINGS.Intern("%s")
	sym_%s.name = string_%s
	v_%s := _ns.Intern(*sym_%s)
`,
			name, *s, name, name, name, name)

		if v_value != "" {
			code += fmt.Sprintf(`
var value_%s = %s
`[1:],
				name, v_value)
			interns += fmt.Sprintf(`
	v_%s.Value = value_%s
`[1:],
				name, name)
		}

		if v_expr != "" {
			code += fmt.Sprintf(`
var expr_%s = %s
`[1:],
				name, v_expr)
			interns += fmt.Sprintf(`
	v_%s.expr = expr_%s
`[1:],
				name, name)
		}
	}

	return code, interns
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

func (pos Position) Emit(env *CodeEnv) string {
	// p = appendInt(p, pos.startLine)
	// p = appendInt(p, pos.endLine)
	// p = appendInt(p, pos.startColumn)
	// p = appendInt(p, pos.endColumn)
	// p = appendUint16(p, env.stringIndex(pos.filename))
	// return p
	return "nil /*Position*/"
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

func (info *ObjectInfo) Emit(env *CodeEnv) string {
	// if info == nil {
	// 	return append(p, NULL)
	// }
	// p = append(p, NOT_NULL)
	// return info.Pos().Emit(p, env)
	return "nil /*ObjectInfo*/"
}

// func unpackObjectInfo(p []byte, header *EmitHeader) (*ObjectInfo, []byte) {
// 	if p[0] == NULL {
// 		return nil, p[1:]
// 	}
// 	p = p[1:]
// 	pos, p := unpackPosition(p, header)
// 	return &ObjectInfo{Position: pos}, p
// }

func EmitObjectOrNull(obj Object, env *CodeEnv) string {
	// if obj == nil {
	// 	return append(p, NULL)
	// }
	// p = append(p, NOT_NULL)
	// return packObject(obj, p, env)
	return "nil /*ObjectOrNull*/"
}

// func UnpackObjectOrNull(p []byte, header *EmitHeader) (Object, []byte) {
// 	if p[0] == NULL {
// 		return nil, p[1:]
// 	}
// 	return unpackObject(p[1:], header)
// }

func (s Symbol) Emit(env *CodeEnv) string {
	// p = s.info.Emit(p, env)
	// p = EmitObjectOrNull(s.meta, p, env)
	// p = appendUint16(p, env.stringIndex(s.name))
	// p = appendUint16(p, env.stringIndex(s.ns))
	// p = appendUint32(p, s.hash)
	// return p
	return "nil /*Symbol*/"
}

// func unpackSymbol(p []byte, header *EmitHeader) (Symbol, []byte) {
// 	info, p := unpackObjectInfo(p, header)
// 	meta, p := UnpackObjectOrNull(p, header)
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

func (t *Type) Emit(env *CodeEnv) string {
	// s := MakeSymbol(t.name)
	// return s.Emit(p, env)
	return "nil /*Type*/"
}

// func unpackType(p []byte, header *EmitHeader) (*Type, []byte) {
// 	s, p := unpackSymbol(p, header)
// 	return TYPES[s.name], p
// }

func emitProc(p Proc, env *CodeEnv) string {
	return "" /* Use direct assignment */
}

func (le *LocalEnv) Emit(env *CodeEnv) string {
	return "nil /**LocalEnv*/"
}

func emitFn(fn *Fn, env *CodeEnv) string {
	fields := []string{}
	if fn.isMacro {
		fields = append(fields, "\tisMacro: true,")
	}
	if fn.fnExpr != nil {
		fields = append(fields, fmt.Sprintf("\tfnExpr: %s,", fn.fnExpr.Emit(env)))
	}
	if fn.env != nil {
		fields = append(fields, fmt.Sprintf("\tenv: %s,", fn.env.Emit(env)))
	}
	f := strings.Join(fields, "\n")
	if f != "" {
		f = "\n" + f + "\n"
	}
	return fmt.Sprintf("&Fn{%s}", f)
}

func emitObject(obj Object, env *CodeEnv) string {
	switch obj := obj.(type) {
	case Symbol:
		return obj.Emit(env)
	case *Var:
		return obj.Emit(env)
	case *Type:
		return obj.Emit(env)
	case Proc:
		return emitProc(obj, env)
	case *Fn:
		return emitFn(obj, env)
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

func (expr *LiteralExpr) Emit(env *CodeEnv) string {
	// p = append(p, LITERAL_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = appendBool(p, expr.isSurrogate)
	// p = packObject(expr.obj, p, env)
	// return p
	return "nil /*LiteralExpr*/"
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

func emitSeq(s []Expr, env *CodeEnv) string {
	// p = appendInt(p, len(s))
	// for _, e := range s {
	// 	p = e.Emit(p, env)
	// }
	// return p
	return "nil /*Seq*/"
}

// func unpackSeq(p []byte, header *EmitHeader) ([]Expr, []byte) {
// 	c, p := extractInt(p)
// 	res := make([]Expr, c)
// 	for i := 0; i < c; i++ {
// 		res[i], p = UnpackExpr(p, header)
// 	}
// 	return res, p
// }

func emitSymbolSeq(s []Symbol, env *CodeEnv) string {
	// p = appendInt(p, len(s))
	// for _, e := range s {
	// 	p = e.Emit(p, env)
	// }
	// return p
	return "nil /*SymbolSeq*/"
}

// func unpackSymbolSeq(p []byte, header *EmitHeader) ([]Symbol, []byte) {
// 	c, p := extractInt(p)
// 	res := make([]Symbol, c)
// 	for i := 0; i < c; i++ {
// 		res[i], p = unpackSymbol(p, header)
// 	}
// 	return res, p
// }

func emitFnArityExprSeq(s []FnArityExpr, env *CodeEnv) string {
	// p = appendInt(p, len(s))
	// for _, e := range s {
	// 	p = e.Emit(p, env)
	// }
	// return p
	return "nil /*FnArityExprSeq*/"
}

// func unpackFnArityExprSeq(p []byte, header *EmitHeader) ([]FnArityExpr, []byte) {
// 	c, p := extractInt(p)
// 	res := make([]FnArityExpr, c)
// 	for i := 0; i < c; i++ {
// 		var e *FnArityExpr
// 		e, p = unpackFnArityExpr(p, header)
// 		res[i] = *e
// 	}
// 	return res, p
// }

func emitCatchExprSeq(s []*CatchExpr, env *CodeEnv) string {
	// p = appendInt(p, len(s))
	// for _, e := range s {
	// 	p = e.Emit(p, env)
	// }
	// return p
	return "nil /*CatchExprSeq*/"
}

// func unpackCatchExprSeq(p []byte, header *EmitHeader) ([]*CatchExpr, []byte) {
// 	c, p := extractInt(p)
// 	res := make([]*CatchExpr, c)
// 	for i := 0; i < c; i++ {
// 		res[i], p = unpackCatchExpr(p, header)
// 	}
// 	return res, p
// }

func (expr *VectorExpr) Emit(env *CodeEnv) string {
	// p = append(p, VECTOR_EXPR)
	// p = expr.Pos().Emit(p, env)
	// return packSeq(p, expr.v, env)
	return "nil /*VectorExpr*/"
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

func (expr *SetExpr) Emit(env *CodeEnv) string {
	// p = append(p, SET_EXPR)
	// p = expr.Pos().Emit(p, env)
	// return packSeq(p, expr.elements, env)
	return "nil /*SetExpr*/"
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

func (expr *MapExpr) Emit(env *CodeEnv) string {
	// p = append(p, MAP_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSeq(p, expr.keys, env)
	// p = packSeq(p, expr.values, env)
	// return p
	return "nil /*MapExpr*/"
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

func (expr *IfExpr) Emit(env *CodeEnv) string {
	// p = append(p, IF_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.cond.Emit(p, env)
	// p = expr.positive.Emit(p, env)
	// p = expr.negative.Emit(p, env)
	// return p
	return "nil /*IfExpr*/"
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

func (expr *DefExpr) Emit(env *CodeEnv) string {
	// p = append(p, DEF_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.name.Emit(p, env)
	// p = EmitExprOrNull(expr.value, p, env)
	// p = EmitExprOrNull(expr.meta, p, env)
	// p = expr.vr.info.Emit(p, env)
	// return p
	if expr.value == nil {
		return "" // just (declare name), which can be ignored here
	}

	name := nameAsGo(*expr.name.name)

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
		expr.Pos().Emit(env),
		expr.vr.Emit(env),
		expr.name.Emit(env),
		EmitExprOrNull(expr.value, env),
		EmitExprOrNull(expr.meta, env))

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
// 	value, p := UnpackExprOrNull(p, header)
// 	meta, p := UnpackExprOrNull(p, header)
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

func (expr *CallExpr) Emit(env *CodeEnv) string {
	// p = append(p, CALL_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.callable.Emit(p, env)
	// p = packSeq(p, expr.args, env)
	// return p
	return "nil /*CallExpr*/"
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

func (expr *RecurExpr) Emit(env *CodeEnv) string {
	// p = append(p, RECUR_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSeq(p, expr.args, env)
	// return p
	return "nil /*RecurExpr*/"
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

func (vr *Var) Emit(env *CodeEnv) string {
	// p = vr.ns.Name.Emit(p, env)
	// p = vr.name.Emit(p, env)
	// return p
	return "nil /*Var*/"
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

func (expr *VarRefExpr) Emit(env *CodeEnv) string {
	// p = append(p, VARREF_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.vr.Emit(p, env)
	// return p
	return "nil /*VarRefExpr*/"
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

func (expr *SetMacroExpr) Emit(env *CodeEnv) string {
	// p = append(p, SET_MACRO_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.vr.Emit(p, env)
	// return p
	return "/*SetMacroExpr*/"
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

func (expr *BindingExpr) Emit(env *CodeEnv) string {
	// p = append(p, BINDING_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = appendInt(p, env.bindingIndex(expr.binding))
	// return p
	return "nil /*BindingExpr*/"
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

func (expr *MetaExpr) Emit(env *CodeEnv) string {
	// p = append(p, META_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.meta.Emit(p, env)
	// p = expr.expr.Emit(p, env)
	// return p
	return "nil /*MetaExpr*/"
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

func (expr *DoExpr) Emit(env *CodeEnv) string {
	// p = append(p, DO_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSeq(p, expr.body, env)
	// return p
	return "/*DoExpr*/"
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

func (expr *FnArityExpr) Emit(env *CodeEnv) string {
	// p = append(p, FN_ARITY_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSymbolSeq(p, expr.args, env)
	// p = packSeq(p, expr.body, env)
	// if expr.taggedType != nil {
	// 	p = append(p, NOT_NULL)
	// 	p = appendUint16(p, env.stringIndex(STRINGS.Intern(expr.taggedType.name)))
	// } else {
	// 	p = append(p, NULL)
	// }
	// return p
	return "nil /*FnArityExpr*/"
}

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

func (expr *FnExpr) Emit(env *CodeEnv) string {
	// p = append(p, FN_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packFnArityExprSeq(p, expr.arities, env)
	// if expr.variadic == nil {
	// 	p = append(p, NULL)
	// } else {
	// 	p = append(p, NOT_NULL)
	// 	p = expr.variadic.Emit(p, env)
	// }
	// p = expr.self.Emit(p, env)
	// return p
	return "nil /*FnExpr*/"
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

func (expr *LetExpr) Emit(env *CodeEnv) string {
	// p = append(p, LET_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSymbolSeq(p, expr.names, env)
	// p = packSeq(p, expr.values, env)
	// p = packSeq(p, expr.body, env)
	// return p
	return "nil /*LetExpr*/"
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

func (expr *LoopExpr) Emit(env *CodeEnv) string {
	// p = append(p, LOOP_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSymbolSeq(p, expr.names, env)
	// p = packSeq(p, expr.values, env)
	// p = packSeq(p, expr.body, env)
	// return p
	return "nil /*LoopExpr*/"
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

func (expr *ThrowExpr) Emit(env *CodeEnv) string {
	// p = append(p, THROW_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = expr.e.Emit(p, env)
	// return p
	return "nil /*ThrowExpr*/"
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

func (expr *CatchExpr) Emit(env *CodeEnv) string {
	// p = append(p, CATCH_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = appendUint16(p, env.stringIndex(STRINGS.Intern(expr.excType.name)))
	// p = expr.excSymbol.Emit(p, env)
	// p = packSeq(p, expr.body, env)
	// return p
	return "nil /*CatchExpr*/"
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

func (expr *TryExpr) Emit(env *CodeEnv) string {
	// p = append(p, TRY_EXPR)
	// p = expr.Pos().Emit(p, env)
	// p = packSeq(p, expr.body, env)
	// p = packCatchExprSeq(p, expr.catches, env)
	// p = packSeq(p, expr.finallyExpr, env)
	// return p
	return "nil /*TryExpr*/"
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

func EmitExprOrNull(expr Expr, env *CodeEnv) string {
	// if expr == nil {
	// 	return append(p, NULL)
	// }
	// p = append(p, NOT_NULL)
	// return expr.Emit(p, env)
	return "nil /*ExprOrNull*/"
}

// func UnpackExprOrNull(p []byte, header *EmitHeader) (Expr, []byte) {
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
