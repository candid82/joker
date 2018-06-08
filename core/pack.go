package core

import "encoding/binary"

const (
	SEQEND       = 0
	LITERAL_EXPR = 1
	VECTOR_EXPR  = 2
	MAP_EXPR     = 3
	INT          = 4
)

type (
	PackEnv struct {
		Strings         map[*string]uint8
		nextStringIndex uint8
	}

	PackHeader struct {
		Strings []string
	}
)

func (env *PackEnv) stringIndex(s *string) uint8 {
	index, ok := env.Strings[s]
	if ok {
		return index
	}
	env.Strings[s] = env.nextStringIndex
	env.nextStringIndex++
	return env.nextStringIndex - 1
}

func (pos Position) Pack(p []byte, env *PackEnv) []byte {
	pp := make([]byte, 8)
	binary.BigEndian.PutUint64(pp, uint64(pos.startLine))
	p = append(p, pp...)
	binary.BigEndian.PutUint64(pp, uint64(pos.endLine))
	p = append(p, pp...)
	binary.BigEndian.PutUint64(pp, uint64(pos.startColumn))
	p = append(p, pp...)
	binary.BigEndian.PutUint64(pp, uint64(pos.endColumn))
	p = append(p, pp...)
	p = append(p, env.stringIndex(pos.filename))
	return p
}

func unpackPosition(p []byte, header *PackHeader) (pos Position, pp []byte) {
	pos.startLine = int(binary.BigEndian.Uint64(p[0:8]))
	pos.endLine = int(binary.BigEndian.Uint64(p[8:16]))
	pos.startColumn = int(binary.BigEndian.Uint64(p[16:24]))
	pos.endColumn = int(binary.BigEndian.Uint64(p[24:32]))
	pos.filename = STRINGS.Intern(header.Strings[p[32]])
	return pos, p[33:]
}

func (i Int) Pack(p []byte) []byte {
	pp := make([]byte, 8)
	binary.BigEndian.PutUint64(pp, uint64(i.I))
	return append(p, pp...)
}

func unpackInt(p []byte) (Int, []byte) {
	return MakeInt(int(binary.BigEndian.Uint64(p[0:8]))), p[8:]
}

func unpackObject(p []byte, header *PackHeader) (Object, []byte) {
	switch p[0] {
	case INT:
		return unpackInt(p[1:])
	default:
		panic(RT.NewError("Unknown pack tag"))
	}
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func byteToBool(b byte) bool {
	if b == 1 {
		return true
	}
	return false
}

func (expr *LiteralExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, LITERAL_EXPR)
	p = expr.Pos().Pack(p, env)
	p = append(p, boolToByte(expr.isSurrogate))
	p = expr.obj.Pack(p)
	return p
}

func unpackLiteralExpr(p []byte, header *PackHeader) (*LiteralExpr, []byte) {
	pos, p := unpackPosition(p, header)
	isSurrogate := byteToBool(p[0])
	obj, p := unpackObject(p[1:], header)
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

func UnpackExpr(p []byte, header *PackHeader) (Expr, []byte) {
	switch p[0] {
	case LITERAL_EXPR:
		return unpackLiteralExpr(p[1:], header)
	case VECTOR_EXPR:
		return unpackVectorExpr(p[1:], header)
	case MAP_EXPR:
		return unpackMapExpr(p[1:], header)
	default:
		panic(RT.NewError("Unknown pack tag"))
	}
}
