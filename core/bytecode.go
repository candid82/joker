package core

import "encoding/binary"

// Numeric operands are unsigned 32-bit integers. Forward and backward jumps
// encode a distance from the end of the instruction, without signed narrowing.
type Opcode uint8

const (
	OP_CONST Opcode = iota // constant index
	OP_NIL
	OP_TRUE
	OP_FALSE
	OP_POP
	OP_DUP
	OP_GET_LOCAL   // frame-relative slot
	OP_SET_LOCAL   // initialize a recursive binding cell
	OP_GET_UPVALUE // function-wide capture index
	OP_BIND_CELL
	OP_GET_VAR // constant index, must contain a Var
	OP_SET_VAR
	OP_SET_VAR_META   // Var constant index, metadata constant index
	OP_MERGE_VAR_META // Var constant index
	OP_FINISH_DEF     // Var constant index
	OP_WITH_META
	OP_CHECK_CALLABLE
	OP_JUMP // forward distance
	OP_JUMP_IF_FALSE
	OP_LOOP    // backward distance
	OP_CALL    // argument count
	OP_CLOSURE // subfunction index; captures are in the subfunction prototype
	OP_RETURN
	OP_RECUR   // argument count, first target slot
	OP_VECTOR  // element count
	OP_MAP_NEW // pair count
	OP_MAP_CHECK
	OP_MAP_ADD
	OP_SET_NEW
	OP_SET_ADD
	OP_POPN // remove N values beneath the result
	OP_THROW
	OP_TRY_BEGIN // handler index
	OP_TRY_END
	OP_FINALLY_END
	OP_SET_MACRO // Var constant index
)

type CatchInfo struct {
	ExcType   *Type
	HandlerIP int
	LocalSlot int
}
type HandlerInfo struct {
	Catches       []CatchInfo
	FinallyIP     int // -1 if absent
	EndIP         int
	TryLocalCount int // stack depth when the handler is installed
}
type Chunk struct {
	Code      []byte
	Constants []Object
	Positions []Position
	callSites map[int]*CallExpr // immutable native-call descriptors
	Handlers  []HandlerInfo
}

func NewChunk() *Chunk {
	return &Chunk{Code: make([]byte, 0, 256), Constants: make([]Object, 0, 16), Positions: make([]Position, 0, 256)}
}
func (c *Chunk) AddHandler(h HandlerInfo) int {
	c.Handlers = append(c.Handlers, h)
	return len(c.Handlers) - 1
}
func (c *Chunk) appendAt(b byte, pos Position) {
	c.Code = append(c.Code, b)
	c.Positions = append(c.Positions, pos)
}
func (c *Chunk) positionAt(ip int) Position {
	if ip >= 0 && ip < len(c.Positions) {
		return c.Positions[ip]
	}
	return Position{}
}
func (c *Chunk) AddConstant(v Object) int {
	c.Constants = append(c.Constants, v)
	return len(c.Constants) - 1
}

type UpvalueInfo struct {
	Index   int
	IsLocal bool
}
type ArityProto struct {
	Arity        int
	IsVariadic   bool
	Chunk        *Chunk
	SubFunctions []*FunctionProto
	ArgTypes     [][]*Type
	TaggedType   *Type
}
type FunctionProto struct {
	Name          string
	Arities       []*ArityProto
	VariadicArity *ArityProto
	Upvalues      []UpvalueInfo // one shared layout for all arities
	// Top-level expressions have a zero-argument body instead of an arity table.
	Chunk        *Chunk
	SubFunctions []*FunctionProto
}

func NewFunctionProto(name string) *FunctionProto {
	return &FunctionProto{Name: name, Chunk: NewChunk()}
}
func (p *FunctionProto) AddSubFunction(sub *FunctionProto) int {
	p.SubFunctions = append(p.SubFunctions, sub)
	return len(p.SubFunctions) - 1
}

var opcodeNames = [...]string{
	OP_CONST: "CONST", OP_NIL: "NIL", OP_TRUE: "TRUE", OP_FALSE: "FALSE", OP_POP: "POP", OP_DUP: "DUP",
	OP_GET_LOCAL: "GET_LOCAL", OP_SET_LOCAL: "SET_LOCAL", OP_GET_UPVALUE: "GET_UPVALUE", OP_BIND_CELL: "BIND_CELL",
	OP_GET_VAR: "GET_VAR", OP_SET_VAR: "SET_VAR", OP_SET_VAR_META: "SET_VAR_META", OP_MERGE_VAR_META: "MERGE_VAR_META", OP_FINISH_DEF: "FINISH_DEF",
	OP_WITH_META: "WITH_META", OP_CHECK_CALLABLE: "CHECK_CALLABLE", OP_JUMP: "JUMP", OP_JUMP_IF_FALSE: "JUMP_IF_FALSE", OP_LOOP: "LOOP",
	OP_CALL: "CALL", OP_CLOSURE: "CLOSURE", OP_RETURN: "RETURN", OP_RECUR: "RECUR", OP_VECTOR: "VECTOR", OP_MAP_NEW: "MAP_NEW",
	OP_MAP_CHECK: "MAP_CHECK", OP_MAP_ADD: "MAP_ADD", OP_SET_NEW: "SET_NEW", OP_SET_ADD: "SET_ADD", OP_POPN: "POPN",
	OP_THROW: "THROW", OP_TRY_BEGIN: "TRY_BEGIN", OP_TRY_END: "TRY_END", OP_FINALLY_END: "FINALLY_END", OP_SET_MACRO: "SET_MACRO",
}

func operandAt(code []byte, offset int) int {
	return int(binary.BigEndian.Uint32(code[offset : offset+4]))
}
func opcodeOperands(op Opcode) (int, bool) {
	switch op {
	case OP_CONST, OP_GET_LOCAL, OP_SET_LOCAL, OP_GET_UPVALUE, OP_GET_VAR, OP_SET_VAR,
		OP_JUMP, OP_JUMP_IF_FALSE, OP_LOOP, OP_CALL, OP_CLOSURE, OP_VECTOR, OP_POPN,
		OP_TRY_BEGIN, OP_MERGE_VAR_META, OP_SET_MACRO, OP_MAP_NEW, OP_FINISH_DEF:
		return 1, true
	case OP_RECUR, OP_SET_VAR_META:
		return 2, true
	default:
		return 0, int(op) < len(opcodeNames)
	}
}
func OpcodeName(op Opcode) string {
	if int(op) < len(opcodeNames) {
		return opcodeNames[op]
	}
	return "UNKNOWN"
}
