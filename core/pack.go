package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	LITERAL_EXPR   = 1
	VECTOR_EXPR    = 2
	MAP_EXPR       = 3
	SET_EXPR       = 4
	IF_EXPR        = 5
	DEF_EXPR       = 6
	CALL_EXPR      = 7
	RECUR_EXPR     = 8
	META_EXPR      = 9
	DO_EXPR        = 10
	FN_ARITY_EXPR  = 11
	FN_EXPR        = 12
	LET_EXPR       = 13
	THROW_EXPR     = 14
	CATCH_EXPR     = 15
	TRY_EXPR       = 16
	VARREF_EXPR    = 17
	BINDING_EXPR   = 18
	LOOP_EXPR      = 19
	SET_MACRO_EXPR = 20
	NULL           = 100
	NOT_NULL       = 101
	SYMBOL_OBJ     = 102
	VAR_OBJ        = 103
	TYPE_OBJ       = 104
)

type (
	PackEnv struct {
		Strings          map[*string]uint16
		Bindings         map[*Binding]int
		nextStringIndex  uint16
		nextBindingIndex int
	}

	PackHeader struct {
		GlobalEnv *Env
		Strings   []*string
		Bindings  []Binding
	}
)

func (b *Binding) Pack(p []byte, env *PackEnv) []byte {
	p = b.name.Pack(p, env)
	p = appendInt(p, b.index)
	p = appendInt(p, b.frame)
	p = appendBool(p, b.isUsed)
	return p
}

func unpackBinding(p []byte, header *PackHeader) (Binding, []byte) {
	name, p := unpackSymbol(p, header)
	index, p := extractInt(p)
	frame, p := extractInt(p)
	isUsed, p := extractBool(p)
	return Binding{
		name:   name,
		index:  index,
		frame:  frame,
		isUsed: isUsed,
	}, p
}

func NewPackEnv() *PackEnv {
	return &PackEnv{
		Strings:  make(map[*string]uint16),
		Bindings: make(map[*Binding]int),
	}
}

func (env *PackEnv) Pack(p []byte) []byte {
	var bp []byte
	bp = appendInt(bp, len(env.Bindings))
	for k, v := range env.Bindings {
		bp = appendInt(bp, v)
		bp = k.Pack(bp, env)
	}
	p = appendInt(p, len(env.Strings))
	for k, v := range env.Strings {
		p = appendUint16(p, v)
		if k == nil {
			p = appendInt(p, -1)
		} else {
			p = appendInt(p, len(*k))
			p = append(p, *k...)
		}
	}
	p = append(p, bp...)
	return p
}

func UnpackHeader(p []byte, env *Env) (*PackHeader, []byte) {
	stringCount, p := extractInt(p)
	strs := make([]*string, stringCount)
	for i := 0; i < stringCount; i++ {
		var index uint16
		var length int
		index, p = extractUInt16(p)
		length, p = extractInt(p)
		if length == -1 {
			strs[index] = nil
		} else {
			strs[index] = STRINGS.Intern(string(p[:length]))
			p = p[length:]
		}
	}
	header := &PackHeader{
		GlobalEnv: env,
		Strings:   strs,
	}
	bindingCount, p := extractInt(p)
	bindings := make([]Binding, bindingCount)
	for i := 0; i < bindingCount; i++ {
		var index int
		var b Binding
		index, p = extractInt(p)
		b, p = unpackBinding(p, header)
		bindings[index] = b
	}
	header.Bindings = bindings
	return header, p
}

func (env *PackEnv) stringIndex(s *string) uint16 {
	index, ok := env.Strings[s]
	if ok {
		return index
	}
	env.Strings[s] = env.nextStringIndex
	env.nextStringIndex++
	return env.nextStringIndex - 1
}

func (env *PackEnv) bindingIndex(b *Binding) int {
	index, ok := env.Bindings[b]
	if ok {
		return index
	}
	env.Bindings[b] = env.nextBindingIndex
	env.nextBindingIndex++
	return env.nextBindingIndex - 1
}

func appendBool(p []byte, b bool) []byte {
	var bb byte
	if b {
		bb = 1
	}
	return append(p, bb)
}

func extractBool(p []byte) (bool, []byte) {
	var b bool
	if p[0] == 1 {
		b = true
	}
	return b, p[1:]
}

func appendUint16(p []byte, i uint16) []byte {
	pp := make([]byte, 2)
	binary.BigEndian.PutUint16(pp, i)
	p = append(p, pp...)
	return p
}

func extractUInt16(p []byte) (uint16, []byte) {
	return binary.BigEndian.Uint16(p[0:2]), p[2:]
}

func appendUint32(p []byte, i uint32) []byte {
	pp := make([]byte, 4)
	binary.BigEndian.PutUint32(pp, i)
	p = append(p, pp...)
	return p
}

func extractUInt32(p []byte) (uint32, []byte) {
	return binary.BigEndian.Uint32(p[0:4]), p[4:]
}

func appendInt(p []byte, i int) []byte {
	pp := make([]byte, 8)
	binary.BigEndian.PutUint64(pp, uint64(i))
	p = append(p, pp...)
	return p
}

func extractInt(p []byte) (int, []byte) {
	return int(binary.BigEndian.Uint64(p[0:8])), p[8:]
}

func (pos Position) Pack(p []byte, env *PackEnv) []byte {
	p = appendInt(p, pos.startLine)
	p = appendInt(p, pos.endLine)
	p = appendInt(p, pos.startColumn)
	p = appendInt(p, pos.endColumn)
	p = appendUint16(p, env.stringIndex(pos.filename))
	return p
}

func unpackPosition(p []byte, header *PackHeader) (pos Position, pp []byte) {
	pos.startLine, p = extractInt(p)
	pos.endLine, p = extractInt(p)
	pos.startColumn, p = extractInt(p)
	pos.endColumn, p = extractInt(p)
	i, p := extractUInt16(p)
	pos.filename = header.Strings[i]
	return pos, p
}

func (info *ObjectInfo) Pack(p []byte, env *PackEnv) []byte {
	if info == nil {
		return append(p, NULL)
	}
	p = append(p, NOT_NULL)
	return info.Pos().Pack(p, env)
}

func unpackObjectInfo(p []byte, header *PackHeader) (*ObjectInfo, []byte) {
	if p[0] == NULL {
		return nil, p[1:]
	}
	p = p[1:]
	pos, p := unpackPosition(p, header)
	return &ObjectInfo{Position: pos}, p
}

func PackObjectOrNull(obj Object, p []byte, env *PackEnv) []byte {
	if obj == nil {
		return append(p, NULL)
	}
	p = append(p, NOT_NULL)
	return packObject(obj, p, env)
}

func UnpackObjectOrNull(p []byte, header *PackHeader) (Object, []byte) {
	if p[0] == NULL {
		return nil, p[1:]
	}
	return unpackObject(p[1:], header)
}

func (s Symbol) Pack(p []byte, env *PackEnv) []byte {
	p = s.info.Pack(p, env)
	p = PackObjectOrNull(s.meta, p, env)
	p = appendUint16(p, env.stringIndex(s.name))
	p = appendUint16(p, env.stringIndex(s.ns))
	p = appendUint32(p, s.hash)
	return p
}

func unpackSymbol(p []byte, header *PackHeader) (Symbol, []byte) {
	info, p := unpackObjectInfo(p, header)
	meta, p := UnpackObjectOrNull(p, header)
	iname, p := extractUInt16(p)
	ins, p := extractUInt16(p)
	hash, p := extractUInt32(p)
	res := Symbol{
		InfoHolder: InfoHolder{info: info},
		name:       header.Strings[iname],
		ns:         header.Strings[ins],
		hash:       hash,
	}
	if meta != nil {
		res.meta = meta.(Map)
	}
	return res, p
}

func (t *Type) Pack(p []byte, env *PackEnv) []byte {
	s := MakeSymbol(t.name)
	return s.Pack(p, env)
}

func unpackType(p []byte, header *PackHeader) (*Type, []byte) {
	s, p := unpackSymbol(p, header)
	return TYPES[s.name], p
}

func packObject(obj Object, p []byte, env *PackEnv) []byte {
	switch obj := obj.(type) {
	case Symbol:
		p = append(p, SYMBOL_OBJ)
		return obj.Pack(p, env)
	case *Var:
		p = append(p, VAR_OBJ)
		p = obj.Pack(p, env)
		return p
	case *Type:
		p = append(p, TYPE_OBJ)
		p = obj.Pack(p, env)
		return p
	default:
		p = append(p, NULL)
		var buf bytes.Buffer
		PrintObject(obj, &buf)
		bb := buf.Bytes()
		p = appendInt(p, len(bb))
		p = append(p, bb...)
		return p
	}
}

func unpackObject(p []byte, header *PackHeader) (Object, []byte) {
	switch p[0] {
	case SYMBOL_OBJ:
		return unpackSymbol(p[1:], header)
	case VAR_OBJ:
		return unpackVar(p[1:], header)
	case TYPE_OBJ:
		return unpackType(p[1:], header)
	case NULL:
		var size int
		size, p = extractInt(p[1:])
		obj := readFromReader(bytes.NewReader(p[:size]))
		return obj, p[size:]
	default:
		panic(RT.NewError(fmt.Sprintf("Unknown object tag: %d", p[0])))
	}
}

func (expr *LiteralExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, LITERAL_EXPR)
	p = expr.Pos().Pack(p, env)
	p = appendBool(p, expr.isSurrogate)
	p = packObject(expr.obj, p, env)
	return p
}

func unpackLiteralExpr(p []byte, header *PackHeader) (*LiteralExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	isSurrogate, p := extractBool(p)
	obj, p := unpackObject(p, header)
	res := &LiteralExpr{
		obj:         obj,
		Position:    pos,
		isSurrogate: isSurrogate,
	}
	return res, p
}

func packSeq(p []byte, s []Expr, env *PackEnv) []byte {
	p = appendInt(p, len(s))
	for _, e := range s {
		p = e.Pack(p, env)
	}
	return p
}

func unpackSeq(p []byte, header *PackHeader) ([]Expr, []byte) {
	c, p := extractInt(p)
	res := make([]Expr, c)
	for i := 0; i < c; i++ {
		res[i], p = UnpackExpr(p, header)
	}
	return res, p
}

func packSymbolSeq(p []byte, s []Symbol, env *PackEnv) []byte {
	p = appendInt(p, len(s))
	for _, e := range s {
		p = e.Pack(p, env)
	}
	return p
}

func unpackSymbolSeq(p []byte, header *PackHeader) ([]Symbol, []byte) {
	c, p := extractInt(p)
	res := make([]Symbol, c)
	for i := 0; i < c; i++ {
		res[i], p = unpackSymbol(p, header)
	}
	return res, p
}

func packFnArityExprSeq(p []byte, s []FnArityExpr, env *PackEnv) []byte {
	p = appendInt(p, len(s))
	for _, e := range s {
		p = e.Pack(p, env)
	}
	return p
}

func unpackFnArityExprSeq(p []byte, header *PackHeader) ([]FnArityExpr, []byte) {
	c, p := extractInt(p)
	res := make([]FnArityExpr, c)
	for i := 0; i < c; i++ {
		var e *FnArityExpr
		e, p = unpackFnArityExpr(p, header)
		res[i] = *e
	}
	return res, p
}

func packCatchExprSeq(p []byte, s []*CatchExpr, env *PackEnv) []byte {
	p = appendInt(p, len(s))
	for _, e := range s {
		p = e.Pack(p, env)
	}
	return p
}

func unpackCatchExprSeq(p []byte, header *PackHeader) ([]*CatchExpr, []byte) {
	c, p := extractInt(p)
	res := make([]*CatchExpr, c)
	for i := 0; i < c; i++ {
		res[i], p = unpackCatchExpr(p, header)
	}
	return res, p
}

func (expr *VectorExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, VECTOR_EXPR)
	p = expr.Pos().Pack(p, env)
	return packSeq(p, expr.v, env)
}

func unpackVectorExpr(p []byte, header *PackHeader) (*VectorExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	v, p := unpackSeq(p, header)
	res := &VectorExpr{
		Position: pos,
		v:        v,
	}
	return res, p
}

func (expr *SetExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, SET_EXPR)
	p = expr.Pos().Pack(p, env)
	return packSeq(p, expr.elements, env)
}

func unpackSetExpr(p []byte, header *PackHeader) (*SetExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	v, p := unpackSeq(p, header)
	res := &SetExpr{
		Position: pos,
		elements: v,
	}
	return res, p
}

func (expr *MapExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, MAP_EXPR)
	p = expr.Pos().Pack(p, env)
	p = packSeq(p, expr.keys, env)
	p = packSeq(p, expr.values, env)
	return p
}

func unpackMapExpr(p []byte, header *PackHeader) (*MapExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	ks, p := unpackSeq(p, header)
	vs, p := unpackSeq(p, header)
	res := &MapExpr{
		Position: pos,
		keys:     ks,
		values:   vs,
	}
	return res, p
}

func (expr *IfExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, IF_EXPR)
	p = expr.Pos().Pack(p, env)
	p = expr.cond.Pack(p, env)
	p = expr.positive.Pack(p, env)
	p = expr.negative.Pack(p, env)
	return p
}

func unpackIfExpr(p []byte, header *PackHeader) (*IfExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	cond, p := UnpackExpr(p, header)
	positive, p := UnpackExpr(p, header)
	negative, p := UnpackExpr(p, header)
	res := &IfExpr{
		Position: pos,
		positive: positive,
		negative: negative,
		cond:     cond,
	}
	return res, p
}

func (expr *DefExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, DEF_EXPR)
	p = expr.Pos().Pack(p, env)
	p = expr.name.Pack(p, env)
	p = PackExprOrNull(expr.value, p, env)
	p = PackExprOrNull(expr.meta, p, env)
	p = expr.vr.info.Pack(p, env)
	return p
}

func unpackDefExpr(p []byte, header *PackHeader) (*DefExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	name, p := unpackSymbol(p, header)
	varName := name
	varName.ns = nil
	vr := header.GlobalEnv.CurrentNamespace().Intern(varName)
	value, p := UnpackExprOrNull(p, header)
	meta, p := UnpackExprOrNull(p, header)
	varInfo, p := unpackObjectInfo(p, header)
	updateVar(vr, varInfo, value, name)
	res := &DefExpr{
		Position: pos,
		vr:       vr,
		name:     name,
		value:    value,
		meta:     meta,
	}
	return res, p
}

func (expr *CallExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, CALL_EXPR)
	p = expr.Pos().Pack(p, env)
	p = expr.callable.Pack(p, env)
	p = packSeq(p, expr.args, env)
	return p
}

func unpackCallExpr(p []byte, header *PackHeader) (*CallExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	callable, p := UnpackExpr(p, header)
	args, p := unpackSeq(p, header)
	res := &CallExpr{
		Position: pos,
		callable: callable,
		args:     args,
	}
	return res, p
}

func (expr *RecurExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, RECUR_EXPR)
	p = expr.Pos().Pack(p, env)
	p = packSeq(p, expr.args, env)
	return p
}

func unpackRecurExpr(p []byte, header *PackHeader) (*RecurExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	args, p := unpackSeq(p, header)
	res := &RecurExpr{
		Position: pos,
		args:     args,
	}
	return res, p
}

func (vr *Var) Pack(p []byte, env *PackEnv) []byte {
	p = vr.ns.Name.Pack(p, env)
	p = vr.name.Pack(p, env)
	return p
}

func unpackVar(p []byte, header *PackHeader) (*Var, []byte) {
	nsName, p := unpackSymbol(p, header)
	name, p := unpackSymbol(p, header)
	vr := GLOBAL_ENV.FindNamespace(nsName).mappings[name.name]
	if vr == nil {
		panic(RT.NewError("Error unpacking var: cannot find var " + *nsName.name + "/" + *name.name))
	}
	return vr, p
}

func (expr *VarRefExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, VARREF_EXPR)
	p = expr.Pos().Pack(p, env)
	p = expr.vr.Pack(p, env)
	return p
}

func unpackVarRefExpr(p []byte, header *PackHeader) (*VarRefExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	vr, p := unpackVar(p, header)
	res := &VarRefExpr{
		Position: pos,
		vr:       vr,
	}
	return res, p
}

func (expr *SetMacroExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, SET_MACRO_EXPR)
	p = expr.Pos().Pack(p, env)
	p = expr.vr.Pack(p, env)
	return p
}

func unpackSetMacroExpr(p []byte, header *PackHeader) (*SetMacroExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	vr, p := unpackVar(p, header)
	res := &SetMacroExpr{
		Position: pos,
		vr:       vr,
	}
	return res, p
}

func (expr *BindingExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, BINDING_EXPR)
	p = expr.Pos().Pack(p, env)
	p = appendInt(p, env.bindingIndex(expr.binding))
	return p
}

func unpackBindingExpr(p []byte, header *PackHeader) (*BindingExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	index, p := extractInt(p)
	res := &BindingExpr{
		Position: pos,
		binding:  &header.Bindings[index],
	}
	return res, p
}

func (expr *MetaExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, META_EXPR)
	p = expr.Pos().Pack(p, env)
	p = expr.meta.Pack(p, env)
	p = expr.expr.Pack(p, env)
	return p
}

func unpackMetaExpr(p []byte, header *PackHeader) (*MetaExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	meta, p := unpackMapExpr(p, header)
	expr, p := UnpackExpr(p, header)
	res := &MetaExpr{
		Position: pos,
		meta:     meta,
		expr:     expr,
	}
	return res, p
}

func (expr *DoExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, DO_EXPR)
	p = expr.Pos().Pack(p, env)
	p = packSeq(p, expr.body, env)
	return p
}

func unpackDoExpr(p []byte, header *PackHeader) (*DoExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	body, p := unpackSeq(p, header)
	res := &DoExpr{
		Position: pos,
		body:     body,
	}
	return res, p
}

func (expr *FnArityExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, FN_ARITY_EXPR)
	p = expr.Pos().Pack(p, env)
	p = packSymbolSeq(p, expr.args, env)
	p = packSeq(p, expr.body, env)
	if expr.taggedType != nil {
		p = append(p, NOT_NULL)
		p = appendUint16(p, env.stringIndex(STRINGS.Intern(expr.taggedType.name)))
	} else {
		p = append(p, NULL)
	}
	return p
}

func unpackFnArityExpr(p []byte, header *PackHeader) (*FnArityExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	args, p := unpackSymbolSeq(p, header)
	body, p := unpackSeq(p, header)
	var taggedType *Type
	if p[0] == NULL {
		p = p[1:]
	} else {
		p = p[1:]
		var i uint16
		i, p = extractUInt16(p)
		taggedType = TYPES[header.Strings[i]]
	}
	res := &FnArityExpr{
		Position:   pos,
		body:       body,
		args:       args,
		taggedType: taggedType,
	}
	return res, p
}

func (expr *FnExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, FN_EXPR)
	p = expr.Pos().Pack(p, env)
	p = packFnArityExprSeq(p, expr.arities, env)
	if expr.variadic == nil {
		p = append(p, NULL)
	} else {
		p = append(p, NOT_NULL)
		p = expr.variadic.Pack(p, env)
	}
	p = expr.self.Pack(p, env)
	return p
}

func unpackFnExpr(p []byte, header *PackHeader) (*FnExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	arities, p := unpackFnArityExprSeq(p, header)
	var variadic *FnArityExpr
	if p[0] == NULL {
		p = p[1:]
	} else {
		p = p[1:]
		variadic, p = unpackFnArityExpr(p, header)
	}
	self, p := unpackSymbol(p, header)
	res := &FnExpr{
		Position: pos,
		arities:  arities,
		variadic: variadic,
		self:     self,
	}
	return res, p
}

func (expr *LetExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, LET_EXPR)
	p = expr.Pos().Pack(p, env)
	p = packSymbolSeq(p, expr.names, env)
	p = packSeq(p, expr.values, env)
	p = packSeq(p, expr.body, env)
	return p
}

func unpackLetExpr(p []byte, header *PackHeader) (*LetExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	names, p := unpackSymbolSeq(p, header)
	values, p := unpackSeq(p, header)
	body, p := unpackSeq(p, header)
	res := &LetExpr{
		Position: pos,
		names:    names,
		values:   values,
		body:     body,
	}
	return res, p
}

func (expr *LoopExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, LOOP_EXPR)
	p = expr.Pos().Pack(p, env)
	p = packSymbolSeq(p, expr.names, env)
	p = packSeq(p, expr.values, env)
	p = packSeq(p, expr.body, env)
	return p
}

func unpackLoopExpr(p []byte, header *PackHeader) (*LoopExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	names, p := unpackSymbolSeq(p, header)
	values, p := unpackSeq(p, header)
	body, p := unpackSeq(p, header)
	res := &LoopExpr{
		Position: pos,
		names:    names,
		values:   values,
		body:     body,
	}
	return res, p
}

func (expr *ThrowExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, THROW_EXPR)
	p = expr.Pos().Pack(p, env)
	p = expr.e.Pack(p, env)
	return p
}

func unpackThrowExpr(p []byte, header *PackHeader) (*ThrowExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	e, p := UnpackExpr(p, header)
	res := &ThrowExpr{
		Position: pos,
		e:        e,
	}
	return res, p
}

func (expr *CatchExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, CATCH_EXPR)
	p = expr.Pos().Pack(p, env)
	p = appendUint16(p, env.stringIndex(STRINGS.Intern(expr.excType.name)))
	p = expr.excSymbol.Pack(p, env)
	p = packSeq(p, expr.body, env)
	return p
}

func unpackCatchExpr(p []byte, header *PackHeader) (*CatchExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	i, p := extractUInt16(p)
	typeName := header.Strings[i]
	excSymbol, p := unpackSymbol(p, header)
	body, p := unpackSeq(p, header)
	res := &CatchExpr{
		Position:  pos,
		excSymbol: excSymbol,
		body:      body,
		excType:   TYPES[typeName],
	}
	return res, p
}

func (expr *TryExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, TRY_EXPR)
	p = expr.Pos().Pack(p, env)
	p = packSeq(p, expr.body, env)
	p = packCatchExprSeq(p, expr.catches, env)
	p = packSeq(p, expr.finallyExpr, env)
	return p
}

func unpackTryExpr(p []byte, header *PackHeader) (*TryExpr, []byte) {
	p = p[1:]
	pos, p := unpackPosition(p, header)
	body, p := unpackSeq(p, header)
	catches, p := unpackCatchExprSeq(p, header)
	finallyExpr, p := unpackSeq(p, header)
	res := &TryExpr{
		Position:    pos,
		body:        body,
		catches:     catches,
		finallyExpr: finallyExpr,
	}
	return res, p
}

func PackExprOrNull(expr Expr, p []byte, env *PackEnv) []byte {
	if expr == nil {
		return append(p, NULL)
	}
	p = append(p, NOT_NULL)
	return expr.Pack(p, env)
}

func UnpackExprOrNull(p []byte, header *PackHeader) (Expr, []byte) {
	if p[0] == NULL {
		return nil, p[1:]
	}
	return UnpackExpr(p[1:], header)
}

func UnpackExpr(p []byte, header *PackHeader) (Expr, []byte) {
	switch p[0] {
	case LITERAL_EXPR:
		return unpackLiteralExpr(p, header)
	case VECTOR_EXPR:
		return unpackVectorExpr(p, header)
	case MAP_EXPR:
		return unpackMapExpr(p, header)
	case SET_EXPR:
		return unpackSetExpr(p, header)
	case IF_EXPR:
		return unpackIfExpr(p, header)
	case DEF_EXPR:
		return unpackDefExpr(p, header)
	case CALL_EXPR:
		return unpackCallExpr(p, header)
	case RECUR_EXPR:
		return unpackRecurExpr(p, header)
	case META_EXPR:
		return unpackMetaExpr(p, header)
	case DO_EXPR:
		return unpackDoExpr(p, header)
	case FN_ARITY_EXPR:
		return unpackFnArityExpr(p, header)
	case FN_EXPR:
		return unpackFnExpr(p, header)
	case LET_EXPR:
		return unpackLetExpr(p, header)
	case LOOP_EXPR:
		return unpackLoopExpr(p, header)
	case THROW_EXPR:
		return unpackThrowExpr(p, header)
	case CATCH_EXPR:
		return unpackCatchExpr(p, header)
	case TRY_EXPR:
		return unpackTryExpr(p, header)
	case VARREF_EXPR:
		return unpackVarRefExpr(p, header)
	case SET_MACRO_EXPR:
		return unpackSetMacroExpr(p, header)
	case BINDING_EXPR:
		return unpackBindingExpr(p, header)
	default:
		panic(RT.NewError(fmt.Sprintf("Unknown pack tag: %d", p[0])))
	}
}
