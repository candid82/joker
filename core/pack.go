package core

import (
	"bytes"
	"encoding/binary"
)

const (
	SEQEND        = 0
	LITERAL_EXPR  = 1
	VECTOR_EXPR   = 2
	MAP_EXPR      = 3
	SET_EXPR      = 4
	IF_EXPR       = 5
	DEF_EXPR      = 6
	CALL_EXPR     = 7
	RECUR_EXPR    = 8
	META_EXPR     = 9
	DO_EXPR       = 10
	FN_ARITY_EXPR = 11
	FN_EXPR       = 12
	LET_EXPR      = 13
	THROW_EXPR    = 14
	CATCH_EXPR    = 15
	TRY_EXPR      = 16
	VARREF_EXPR   = 17
	BINDING_EXPR  = 18
	LOOP_EXPR     = 19
	NULL          = 100
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
		Binding   []Binding
	}
)

func NewPackEnv() *PackEnv {
	return &PackEnv{
		Strings:  make(map[*string]uint16),
		Bindings: make(map[*Binding]int),
	}
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

func (i Int) Pack(p []byte, env *PackEnv) []byte {
	p = i.info.Pack(p, env)
	p = appendInt(p, i.I)
	return p
}

func unpackInt(p []byte) (Int, []byte) {
	i, p := extractInt(p)
	return MakeInt(i), p
}

func (info *ObjectInfo) Pack(p []byte, env *PackEnv) []byte {
	if info == nil {
		return append(p, NULL)
	}
	return info.Pos().Pack(p, env)
}

func unpackObjectInfo(p []byte, header *PackHeader) (*ObjectInfo, []byte) {
	if p[0] == NULL {
		return nil, p[1:]
	}
	pos, p := unpackPosition(p, header)
	return &ObjectInfo{Position: pos}, p
}

func (s Symbol) Pack(p []byte, env *PackEnv) []byte {
	p = s.info.Pack(p, env)
	p = appendUint16(p, env.stringIndex(s.name))
	p = appendUint16(p, env.stringIndex(s.ns))
	p = appendUint32(p, s.hash)
	return p
}

func unpackSymbol(p []byte, header *PackHeader) (Symbol, []byte) {
	info, p := unpackObjectInfo(p, header)
	iname, p := extractUInt16(p)
	ins, p := extractUInt16(p)
	hash, p := extractUInt32(p)
	res := Symbol{
		InfoHolder: InfoHolder{info: info},
		name:       header.Strings[iname],
		ns:         header.Strings[ins],
		hash:       hash,
	}
	return res, p
}

func packObject(obj Object, p []byte) []byte {
	var buf bytes.Buffer
	printObject(obj, &buf)
	bb := buf.Bytes()
	p = appendInt(p, len(bb))
	p = append(p, bb...)
	return p
}

func unpackObject(p []byte) (Object, []byte) {
	size, p := extractInt(p)
	obj := readFromReader(bytes.NewReader(p[4 : size+4]))
	return obj, p[size+4:]
}

func (expr *LiteralExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, LITERAL_EXPR)
	p = expr.Pos().Pack(p, env)
	p = appendBool(p, expr.isSurrogate)
	p = packObject(expr.obj, p)
	return p
}

func unpackLiteralExpr(p []byte, header *PackHeader) (*LiteralExpr, []byte) {
	pos, p := unpackPosition(p, header)
	isSurrogate, p := extractBool(p)
	obj, p := unpackObject(p)
	res := &LiteralExpr{
		obj:         obj,
		Position:    pos,
		isSurrogate: isSurrogate,
	}
	return res, p
}

func packSeq(p []byte, s []Expr, env *PackEnv) []byte {
	for _, e := range s {
		p = e.Pack(p, env)
	}
	p = append(p, SEQEND)
	return p
}

func unpackSeq(p []byte, header *PackHeader) ([]Expr, []byte) {
	var res []Expr
	for p[0] != SEQEND {
		var e Expr
		e, p = UnpackExpr(p, header)
		res = append(res, e)
	}
	return res, p[1:]
}

func packSymbolSeq(p []byte, s []Symbol, env *PackEnv) []byte {
	for _, e := range s {
		p = e.Pack(p, env)
	}
	p = append(p, SEQEND)
	return p
}

func unpackSymbolSeq(p []byte, header *PackHeader) ([]Symbol, []byte) {
	var res []Symbol
	for p[0] != SEQEND {
		var e Symbol
		e, p = unpackSymbol(p, header)
		res = append(res, e)
	}
	return res, p[1:]
}

func packFnArityExprSeq(p []byte, s []FnArityExpr, env *PackEnv) []byte {
	for _, e := range s {
		p = e.Pack(p, env)
	}
	p = append(p, SEQEND)
	return p
}

func unpackFnArityExprSeq(p []byte, header *PackHeader) ([]FnArityExpr, []byte) {
	var res []FnArityExpr
	for p[0] != SEQEND {
		var e *FnArityExpr
		e, p = unpackFnArityExpr(p, header)
		res = append(res, *e)
	}
	return res, p[1:]
}

func packCatchExprSeq(p []byte, s []*CatchExpr, env *PackEnv) []byte {
	for _, e := range s {
		p = e.Pack(p, env)
	}
	p = append(p, SEQEND)
	return p
}

func unpackCatchExprSeq(p []byte, header *PackHeader) ([]*CatchExpr, []byte) {
	var res []*CatchExpr
	for p[0] != SEQEND {
		var e *CatchExpr
		e, p = unpackCatchExpr(p, header)
		res = append(res, e)
	}
	return res, p[1:]
}

func (expr *VectorExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, VECTOR_EXPR)
	p = expr.Pos().Pack(p, env)
	return packSeq(p, expr.v, env)
}

func unpackVectorExpr(p []byte, header *PackHeader) (*VectorExpr, []byte) {
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
	p = PackExpr(expr.value, p, env)
	p = PackExpr(expr.meta, p, env)
	p = expr.vr.info.Pack(p, env)
	return p
}

func unpackDefExpr(p []byte, header *PackHeader) (*DefExpr, []byte) {
	pos, p := unpackPosition(p, header)
	name, p := unpackSymbol(p, header)
	value, p := UnpackExpr(p, header)
	meta, p := UnpackExpr(p, header)
	varInfo, p := unpackObjectInfo(p, header)
	varName := name
	varName.ns = nil
	vr := header.GlobalEnv.CurrentNamespace().Intern(varName)
	vr.WithInfo(varInfo)
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
	pos, p := unpackPosition(p, header)
	args, p := unpackSeq(p, header)
	res := &RecurExpr{
		Position: pos,
		args:     args,
	}
	return res, p
}

func (expr *VarRefExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, VARREF_EXPR)
	p = expr.Pos().Pack(p, env)
	p = expr.vr.ns.Name.Pack(p, env)
	p = expr.vr.name.Pack(p, env)
	return p
}

func unpackVarRefExpr(p []byte, header *PackHeader) (*VarRefExpr, []byte) {
	pos, p := unpackPosition(p, header)
	nsName, p := unpackSymbol(p, header)
	name, p := unpackSymbol(p, header)
	res := &VarRefExpr{
		Position: pos,
		vr:       GLOBAL_ENV.FindNamespace(nsName).mappings[name.name],
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
	pos, p := unpackPosition(p, header)
	index, p := extractInt(p)
	res := &BindingExpr{
		Position: pos,
		binding:  &header.Binding[index],
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
	return p
}

func unpackFnArityExpr(p []byte, header *PackHeader) (*FnArityExpr, []byte) {
	pos, p := unpackPosition(p, header)
	args, p := unpackSymbolSeq(p, header)
	body, p := unpackSeq(p, header)
	res := &FnArityExpr{
		Position: pos,
		body:     body,
		args:     args,
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
		p = expr.variadic.Pack(p, env)
	}
	p = expr.self.Pack(p, env)
	return p
}

func unpackFnExpr(p []byte, header *PackHeader) (*FnExpr, []byte) {
	pos, p := unpackPosition(p, header)
	arities, p := unpackFnArityExprSeq(p, header)
	var variadic *FnArityExpr
	if p[0] == NULL {
		p = p[1:]
	} else {
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

func PackExpr(expr Expr, p []byte, env *PackEnv) []byte {
	if expr == nil {
		return append(p, NULL)
	}
	return expr.Pack(p, env)
}

func UnpackExpr(p []byte, header *PackHeader) (Expr, []byte) {
	switch p[0] {
	case LITERAL_EXPR:
		return unpackLiteralExpr(p[1:], header)
	case VECTOR_EXPR:
		return unpackVectorExpr(p[1:], header)
	case MAP_EXPR:
		return unpackMapExpr(p[1:], header)
	case SET_EXPR:
		return unpackSetExpr(p[1:], header)
	case IF_EXPR:
		return unpackIfExpr(p[1:], header)
	case DEF_EXPR:
		return unpackDefExpr(p[1:], header)
	case CALL_EXPR:
		return unpackCallExpr(p[1:], header)
	case RECUR_EXPR:
		return unpackRecurExpr(p[1:], header)
	case META_EXPR:
		return unpackMetaExpr(p[1:], header)
	case DO_EXPR:
		return unpackDoExpr(p[1:], header)
	case FN_ARITY_EXPR:
		return unpackFnArityExpr(p[1:], header)
	case FN_EXPR:
		return unpackFnExpr(p[1:], header)
	case LET_EXPR:
		return unpackLetExpr(p[1:], header)
	case LOOP_EXPR:
		return unpackLoopExpr(p[1:], header)
	case THROW_EXPR:
		return unpackThrowExpr(p[1:], header)
	case CATCH_EXPR:
		return unpackCatchExpr(p[1:], header)
	case TRY_EXPR:
		return unpackTryExpr(p[1:], header)
	case VARREF_EXPR:
		return unpackVarRefExpr(p[1:], header)
	case BINDING_EXPR:
		return unpackBindingExpr(p[1:], header)
	case NULL:
		return nil, p[1:]
	default:
		panic(RT.NewError("Unknown pack tag"))
	}
}
