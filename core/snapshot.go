package core

import "encoding/binary"

const (
	LITERAL byte = iota
	INT     byte = iota
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

func (expr *LiteralExpr) Pack(p []byte, env *PackEnv) []byte {
	p = append(p, LITERAL)
	p = expr.Pos().Pack(p, env)
	p = expr.obj.Pack(p)
	return p
}

func unpackLiteral(p []byte, header *PackHeader) *LiteralExpr {
	pos, p := unpackPosition(p, header)
	obj, p := unpackObject(p, header)
	return &LiteralExpr{
		obj:      obj,
		Position: pos,
	}
}

func UnpackExpr(p []byte, header *PackHeader) Expr {
	switch p[0] {
	case LITERAL:
		return unpackLiteral(p[1:], header)
	default:
		panic(RT.NewError("Unknown pack tag"))
	}
}
