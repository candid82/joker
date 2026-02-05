package core

// Opcode represents a single bytecode instruction.
type Opcode uint8

const (
	// Constants
	OP_CONST Opcode = iota // Push constant from pool (2-byte index)
	OP_NIL                 // Push nil
	OP_TRUE                // Push true
	OP_FALSE               // Push false

	// Stack operations
	OP_POP // Pop and discard top of stack
	OP_DUP // Duplicate top of stack

	// Local variable access
	OP_GET_LOCAL // Push local variable (1-byte index)
	OP_SET_LOCAL // Set local variable (1-byte index)

	// Upvalue (closure) access
	OP_GET_UPVALUE // Push captured variable (1-byte index)
	OP_SET_UPVALUE // Set captured variable (1-byte index)

	// Global/Var access
	OP_GET_VAR // Push var value (2-byte constant pool index -> Var)
	OP_SET_VAR // Set var value (2-byte constant pool index -> Var)

	// Arithmetic
	OP_ADD      // Pop 2, push sum
	OP_SUBTRACT // Pop 2, push difference
	OP_MULTIPLY // Pop 2, push product
	OP_DIVIDE   // Pop 2, push quotient
	OP_NEGATE   // Pop 1, push negation

	// Comparison
	OP_EQUAL   // Pop 2, push equality result
	OP_LESS    // Pop 2, push less-than result
	OP_GREATER // Pop 2, push greater-than result

	// Logical
	OP_NOT // Pop 1, push logical negation

	// Control flow
	OP_JUMP          // Unconditional jump (2-byte signed offset)
	OP_JUMP_IF_FALSE // Pop, jump if falsy (2-byte signed offset)
	OP_LOOP          // Jump backward (2-byte unsigned offset)

	// Function operations
	OP_CALL    // Call function (1-byte arg count)
	OP_CLOSURE // Create closure (2-byte constant index, then upvalue info)
	OP_RETURN  // Return from function

	// Tail recursion
	OP_RECUR // Recur with N args (1-byte arg count)

	// Collection constructors
	OP_VECTOR // Create vector (2-byte element count)
	OP_MAP    // Create map (2-byte pair count)
	OP_SET    // Create set (2-byte element count)

	// Upvalue management
	OP_CLOSE_UPVALUE // Close upvalue at stack top

	// Stack manipulation
	OP_POPN // Pop N values from under the top (preserves top)

	// Exception handling
	OP_THROW     // Pop Error from stack and panic
	OP_TRY_BEGIN // Begin try block (2-byte handler info index)
	OP_TRY_END   // End try block (normal exit)
	OP_POP_SLOT  // Remove value at specific slot (1-byte slot index), shift values above down

	// Var metadata
	OP_SET_VAR_META   // Set var meta from constant (2-byte var index, 2-byte meta index)
	OP_MERGE_VAR_META // Merge map from stack into var meta (2-byte var index)
)

// CatchInfo describes one catch clause for exception handling.
type CatchInfo struct {
	ExcType   *Type // Type to catch
	HandlerIP int   // IP of catch body
	LocalSlot int   // Slot for exception binding
}

// HandlerInfo describes a try block's exception handling configuration.
type HandlerInfo struct {
	Catches       []CatchInfo // Catch clauses in order
	FinallyIP     int         // IP of finally block (-1 if none)
	EndIP         int         // IP after finally block (for normal exit)
	TryLocalCount int         // Local count when try block begins (for stack management)
}

// Chunk holds compiled bytecode and associated data.
type Chunk struct {
	Code      []byte        // Bytecode instructions
	Constants []Object      // Constant pool
	Lines     []int         // Line number for each byte (for error reporting)
	Handlers  []HandlerInfo // Exception handler info (indexed by OP_TRY_BEGIN operand)
}

// NewChunk creates a new empty chunk.
func NewChunk() *Chunk {
	return &Chunk{
		Code:      make([]byte, 0, 256),
		Constants: make([]Object, 0, 16),
		Lines:     make([]int, 0, 256),
		Handlers:  make([]HandlerInfo, 0, 4),
	}
}

// AddHandler adds an exception handler and returns its index.
func (c *Chunk) AddHandler(handler HandlerInfo) int {
	c.Handlers = append(c.Handlers, handler)
	return len(c.Handlers) - 1
}

// AppendByte appends a byte to the chunk.
func (c *Chunk) AppendByte(b byte, line int) {
	c.Code = append(c.Code, b)
	c.Lines = append(c.Lines, line)
}

// WriteOp appends an opcode to the chunk.
func (c *Chunk) WriteOp(op Opcode, line int) {
	c.AppendByte(byte(op), line)
}

// AddConstant adds a constant to the pool and returns its index.
func (c *Chunk) AddConstant(value Object) int {
	c.Constants = append(c.Constants, value)
	return len(c.Constants) - 1
}

// WriteConstant writes an OP_CONST instruction with the constant index.
func (c *Chunk) WriteConstant(value Object, line int) {
	idx := c.AddConstant(value)
	c.WriteOp(OP_CONST, line)
	c.AppendByte(byte(idx>>8), line)
	c.AppendByte(byte(idx&0xff), line)
}

// WriteShort writes a 2-byte value (big-endian).
func (c *Chunk) WriteShort(value uint16, line int) {
	c.AppendByte(byte(value>>8), line)
	c.AppendByte(byte(value&0xff), line)
}

// UpvalueInfo describes how to capture an upvalue.
type UpvalueInfo struct {
	Index   uint8 // Index in parent's locals or upvalues
	IsLocal bool  // True if capturing parent's local, false if parent's upvalue
}

// ArityProto holds compiled bytecode for a single function arity.
type ArityProto struct {
	Arity        int              // Number of fixed parameters (excluding rest param)
	IsVariadic   bool             // True if this arity accepts rest args
	Chunk        *Chunk           // The bytecode for this arity
	Upvalues     []UpvalueInfo    // Upvalue capture info for this arity
	SubFunctions []*FunctionProto // Nested function prototypes for this arity
}

// FunctionProto holds a compiled function's bytecode.
type FunctionProto struct {
	Name          string           // Function name (for debugging/errors)
	Arities       []*ArityProto    // Fixed-arity implementations
	VariadicArity *ArityProto      // Variadic implementation (nil if none)
	SubFunctions  []*FunctionProto // Nested function prototypes (for closures)

	// Legacy fields for single-arity (backward compat during transition)
	Arity    int           // Number of required parameters
	Variadic bool          // Whether function accepts rest args
	Chunk    *Chunk        // The bytecode
	Upvalues []UpvalueInfo // Upvalue capture info
}

// NewFunctionProto creates a new function prototype.
func NewFunctionProto(name string) *FunctionProto {
	return &FunctionProto{
		Name:         name,
		Chunk:        NewChunk(),
		Upvalues:     make([]UpvalueInfo, 0, 4),
		SubFunctions: make([]*FunctionProto, 0, 4),
	}
}

// AddSubFunction adds a nested function prototype and returns its index.
func (p *FunctionProto) AddSubFunction(sub *FunctionProto) int {
	p.SubFunctions = append(p.SubFunctions, sub)
	return len(p.SubFunctions) - 1
}

// Opcode names for debugging/disassembly.
var opcodeNames = [...]string{
	OP_CONST:         "CONST",
	OP_NIL:           "NIL",
	OP_TRUE:          "TRUE",
	OP_FALSE:         "FALSE",
	OP_POP:           "POP",
	OP_DUP:           "DUP",
	OP_GET_LOCAL:     "GET_LOCAL",
	OP_SET_LOCAL:     "SET_LOCAL",
	OP_GET_UPVALUE:   "GET_UPVALUE",
	OP_SET_UPVALUE:   "SET_UPVALUE",
	OP_GET_VAR:       "GET_VAR",
	OP_SET_VAR:       "SET_VAR",
	OP_ADD:           "ADD",
	OP_SUBTRACT:      "SUBTRACT",
	OP_MULTIPLY:      "MULTIPLY",
	OP_DIVIDE:        "DIVIDE",
	OP_NEGATE:        "NEGATE",
	OP_EQUAL:         "EQUAL",
	OP_LESS:          "LESS",
	OP_GREATER:       "GREATER",
	OP_NOT:           "NOT",
	OP_JUMP:          "JUMP",
	OP_JUMP_IF_FALSE: "JUMP_IF_FALSE",
	OP_LOOP:          "LOOP",
	OP_CALL:          "CALL",
	OP_CLOSURE:       "CLOSURE",
	OP_RETURN:        "RETURN",
	OP_RECUR:         "RECUR",
	OP_VECTOR:        "VECTOR",
	OP_MAP:           "MAP",
	OP_SET:           "SET",
	OP_CLOSE_UPVALUE: "CLOSE_UPVALUE",
	OP_POPN:          "POPN",
	OP_THROW:         "THROW",
	OP_TRY_BEGIN:     "TRY_BEGIN",
	OP_TRY_END:       "TRY_END",
}

// OpcodeName returns the name of an opcode.
func OpcodeName(op Opcode) string {
	if int(op) < len(opcodeNames) {
		return opcodeNames[op]
	}
	return "UNKNOWN"
}
