package core

import (
	"strconv"
)

// Local represents a local variable in the compiler.
type Local struct {
	name     Symbol
	depth    int  // Scope depth
	captured bool // Whether captured by a closure
}

// Compiler compiles expressions to bytecode.
type Compiler struct {
	enclosing  *Compiler       // For nested functions
	function   *FunctionProto  // Function being compiled
	locals     []Local         // Local variables in scope
	localCount int             // Number of locals
	scopeDepth int             // Current scope depth
	loopStart  int             // Bytecode offset of loop start (for recur)
	loopDepth  int             // Scope depth at loop start
}

// NewCompiler creates a new compiler for a function.
func NewCompiler(enclosing *Compiler, name string) *Compiler {
	c := &Compiler{
		enclosing: enclosing,
		function:  NewFunctionProto(name),
		locals:    make([]Local, 256),
		loopStart: -1,
	}
	// Reserve slot 0 for the function itself
	c.locals[0] = Local{depth: 0}
	c.localCount = 1
	return c
}

// Compile compiles an expression and returns a function prototype.
func Compile(expr Expr, name string) (*FunctionProto, error) {
	c := NewCompiler(nil, name)
	if err := c.compile(expr); err != nil {
		return nil, err
	}
	c.emitReturn()
	return c.function, nil
}

// CompileFnExpr compiles a function expression.
func CompileFnExpr(fnExpr *FnExpr, env *LocalEnv) (*FunctionProto, error) {
	name := "<anonymous>"
	if fnExpr.self.name != nil {
		name = fnExpr.self.Name()
	}

	// For now, only compile single-arity non-variadic functions
	if len(fnExpr.arities) != 1 || fnExpr.variadic != nil {
		return nil, RT.NewError("VM only supports single-arity non-variadic functions currently")
	}

	return CompileFnArity(fnExpr.arities[0], name)
}

// CompileFnArity compiles a single function arity to bytecode.
func CompileFnArity(arity FnArityExpr, name string) (*FunctionProto, error) {
	c := NewCompiler(nil, name)
	c.function.Arity = len(arity.args)

	// Add parameters as locals
	for _, arg := range arity.args {
		c.addLocal(arg)
	}

	// Compile the body
	for i, bodyExpr := range arity.body {
		if err := c.compile(bodyExpr); err != nil {
			return nil, err
		}
		// Pop intermediate results except the last
		if i < len(arity.body)-1 {
			c.emitOp(OP_POP)
		}
	}
	if len(arity.body) == 0 {
		c.emitOp(OP_NIL)
	}

	c.emitReturn()
	return c.function, nil
}

// compile compiles a single expression.
func (c *Compiler) compile(expr Expr) error {
	switch e := expr.(type) {
	case *LiteralExpr:
		return c.compileLiteral(e)
	case *VectorExpr:
		return c.compileVector(e)
	case *MapExpr:
		return c.compileMap(e)
	case *SetExpr:
		return c.compileSet(e)
	case *IfExpr:
		return c.compileIf(e)
	case *DoExpr:
		return c.compileDo(e)
	case *LetExpr:
		return c.compileLet(e)
	case *LoopExpr:
		return c.compileLoop(e)
	case *RecurExpr:
		return c.compileRecur(e)
	case *VarRefExpr:
		return c.compileVarRef(e)
	case *BindingExpr:
		return c.compileBinding(e)
	case *CallExpr:
		return c.compileCall(e)
	case *DefExpr:
		return c.compileDef(e)
	case *FnExpr:
		return c.compileFn(e)
	case *MetaExpr:
		return c.compile(e.expr)
	case *ThrowExpr:
		return c.compileThrow(e)
	default:
		return RT.NewError("Cannot compile expression type: " + expr.Dump(false).ToString(false))
	}
}

func (c *Compiler) compileLiteral(e *LiteralExpr) error {
	obj := e.obj
	switch v := obj.(type) {
	case Nil:
		c.emitOp(OP_NIL)
	case Boolean:
		if v.B {
			c.emitOp(OP_TRUE)
		} else {
			c.emitOp(OP_FALSE)
		}
	default:
		c.emitConstant(obj)
	}
	return nil
}

func (c *Compiler) compileVector(e *VectorExpr) error {
	for _, elem := range e.v {
		if err := c.compile(elem); err != nil {
			return err
		}
	}
	c.emitOp(OP_VECTOR)
	c.emitShort(uint16(len(e.v)))
	return nil
}

func (c *Compiler) compileMap(e *MapExpr) error {
	for i := range e.keys {
		if err := c.compile(e.keys[i]); err != nil {
			return err
		}
		if err := c.compile(e.values[i]); err != nil {
			return err
		}
	}
	c.emitOp(OP_MAP)
	c.emitShort(uint16(len(e.keys)))
	return nil
}

func (c *Compiler) compileSet(e *SetExpr) error {
	for _, elem := range e.elements {
		if err := c.compile(elem); err != nil {
			return err
		}
	}
	c.emitOp(OP_SET)
	c.emitShort(uint16(len(e.elements)))
	return nil
}

func (c *Compiler) compileIf(e *IfExpr) error {
	// Compile condition
	if err := c.compile(e.cond); err != nil {
		return err
	}

	// Jump to else branch if false
	elseJump := c.emitJump(OP_JUMP_IF_FALSE)

	// Compile then branch
	if err := c.compile(e.positive); err != nil {
		return err
	}

	// Jump over else branch
	endJump := c.emitJump(OP_JUMP)

	// Patch else jump
	c.patchJump(elseJump)

	// Compile else branch
	if err := c.compile(e.negative); err != nil {
		return err
	}

	// Patch end jump
	c.patchJump(endJump)
	return nil
}

func (c *Compiler) compileDo(e *DoExpr) error {
	for i, bodyExpr := range e.body {
		if err := c.compile(bodyExpr); err != nil {
			return err
		}
		// Pop intermediate results except the last
		if i < len(e.body)-1 {
			c.emitOp(OP_POP)
		}
	}
	// If body is empty, push nil
	if len(e.body) == 0 {
		c.emitOp(OP_NIL)
	}
	return nil
}

func (c *Compiler) compileLet(e *LetExpr) error {
	c.beginScope()

	// Compile bindings
	for i, name := range e.names {
		if err := c.compile(e.values[i]); err != nil {
			return err
		}
		c.addLocal(name)
	}

	// Compile body
	for i, bodyExpr := range e.body {
		if err := c.compile(bodyExpr); err != nil {
			return err
		}
		if i < len(e.body)-1 {
			c.emitOp(OP_POP)
		}
	}
	if len(e.body) == 0 {
		c.emitOp(OP_NIL)
	}

	c.endScope()
	return nil
}

func (c *Compiler) compileLoop(e *LoopExpr) error {
	c.beginScope()

	// Save previous loop state
	prevLoopStart := c.loopStart
	prevLoopDepth := c.loopDepth

	// Compile bindings
	for i, name := range e.names {
		if err := c.compile(e.values[i]); err != nil {
			return err
		}
		c.addLocal(name)
	}

	// Mark loop start
	c.loopStart = len(c.function.Chunk.Code)
	c.loopDepth = c.scopeDepth

	// Compile body
	for i, bodyExpr := range e.body {
		if err := c.compile(bodyExpr); err != nil {
			return err
		}
		if i < len(e.body)-1 {
			c.emitOp(OP_POP)
		}
	}
	if len(e.body) == 0 {
		c.emitOp(OP_NIL)
	}

	// Restore loop state
	c.loopStart = prevLoopStart
	c.loopDepth = prevLoopDepth

	c.endScope()
	return nil
}

func (c *Compiler) compileRecur(e *RecurExpr) error {
	if c.loopStart < 0 {
		return RT.NewError("recur outside of loop")
	}

	// Compile new argument values
	for _, arg := range e.args {
		if err := c.compile(arg); err != nil {
			return err
		}
	}

	// Emit recur instruction
	c.emitOp(OP_RECUR)
	c.emitByte(byte(len(e.args)))

	// Jump back to loop start
	offset := len(c.function.Chunk.Code) - c.loopStart + 3 // +3 for the LOOP instruction
	c.emitOp(OP_LOOP)
	c.emitShort(uint16(offset))

	return nil
}

func (c *Compiler) compileVarRef(e *VarRefExpr) error {
	idx := c.function.Chunk.AddConstant(e.vr)
	if idx > 65535 {
		panic(RT.NewError("Too many constants in function"))
	}
	c.emitOp(OP_GET_VAR)
	c.emitShort(uint16(idx))
	return nil
}

func (c *Compiler) compileBinding(e *BindingExpr) error {
	// Find the local variable
	idx := c.resolveLocal(e.binding.name)
	if idx >= 0 {
		c.emitOp(OP_GET_LOCAL)
		c.emitByte(byte(idx))
		return nil
	}

	// Check for upvalue
	idx = c.resolveUpvalue(e.binding.name)
	if idx >= 0 {
		c.emitOp(OP_GET_UPVALUE)
		c.emitByte(byte(idx))
		return nil
	}

	return RT.NewError("Cannot resolve binding: " + e.binding.name.ToString(false))
}

func (c *Compiler) compileCall(e *CallExpr) error {
	// Check for special intrinsic operations
	if varRef, ok := e.callable.(*VarRefExpr); ok {
		if handled, err := c.compileIntrinsic(varRef.vr, e.args); handled {
			return err
		}
	}

	// Compile the callable
	if err := c.compile(e.callable); err != nil {
		return err
	}

	// Compile arguments
	for _, arg := range e.args {
		if err := c.compile(arg); err != nil {
			return err
		}
	}

	// Emit call
	c.emitOp(OP_CALL)
	c.emitByte(byte(len(e.args)))
	return nil
}

// compileIntrinsic tries to compile built-in operations as direct bytecode.
// Returns (handled, error) - if handled is true, the call was compiled as an intrinsic.
func (c *Compiler) compileIntrinsic(v *Var, args []Expr) (bool, error) {
	if v.ns == nil || v.ns.Name.Name() != "joker.core" {
		return false, nil
	}

	name := v.name.Name()
	switch name {
	case "+", "add__":
		return c.compileBinaryOp(args, OP_ADD)
	case "-", "subtract__":
		if len(args) == 1 {
			// Unary negation
			if err := c.compile(args[0]); err != nil {
				return true, err
			}
			c.emitOp(OP_NEGATE)
			return true, nil
		}
		return c.compileBinaryOp(args, OP_SUBTRACT)
	case "*", "multiply__":
		return c.compileBinaryOp(args, OP_MULTIPLY)
	case "/", "divide__":
		return c.compileBinaryOp(args, OP_DIVIDE)
	case "<", "<__":
		return c.compileBinaryOp(args, OP_LESS)
	case ">", ">__":
		return c.compileBinaryOp(args, OP_GREATER)
	case "=":
		return c.compileBinaryOp(args, OP_EQUAL)
	case "not":
		if len(args) != 1 {
			return false, nil
		}
		if err := c.compile(args[0]); err != nil {
			return true, err
		}
		c.emitOp(OP_NOT)
		return true, nil
	}

	return false, nil
}

func (c *Compiler) compileBinaryOp(args []Expr, op Opcode) (bool, error) {
	if len(args) != 2 {
		return false, nil
	}
	if err := c.compile(args[0]); err != nil {
		return true, err
	}
	if err := c.compile(args[1]); err != nil {
		return true, err
	}
	c.emitOp(op)
	return true, nil
}

func (c *Compiler) compileDef(e *DefExpr) error {
	// Compile the value
	if e.value != nil {
		if err := c.compile(e.value); err != nil {
			return err
		}
	} else {
		c.emitOp(OP_NIL)
	}

	// Store in the var
	idx := c.function.Chunk.AddConstant(e.vr)
	c.emitOp(OP_SET_VAR)
	c.emitShort(uint16(idx))

	// Push the var as the result
	c.emitConstant(e.vr)
	return nil
}

func (c *Compiler) compileFn(e *FnExpr) error {
	// Create a new compiler for the nested function
	name := "<anonymous>"
	if e.self.name != nil {
		name = e.self.Name()
	}

	// For now, only compile single-arity non-variadic functions
	if len(e.arities) != 1 || e.variadic != nil {
		return RT.NewError("VM only supports single-arity non-variadic functions currently")
	}

	arity := e.arities[0]
	inner := NewCompiler(c, name)
	inner.function.Arity = len(arity.args)

	// Add parameters as locals
	for _, arg := range arity.args {
		inner.addLocal(arg)
	}

	// Compile the body
	for i, bodyExpr := range arity.body {
		if err := inner.compile(bodyExpr); err != nil {
			return err
		}
		if i < len(arity.body)-1 {
			inner.emitOp(OP_POP)
		}
	}
	if len(arity.body) == 0 {
		inner.emitOp(OP_NIL)
	}

	inner.emitReturn()

	// Add the inner function to our sub-functions
	idx := c.function.AddSubFunction(inner.function)

	// Emit closure instruction
	c.emitOp(OP_CLOSURE)
	c.emitShort(uint16(idx))

	// Emit upvalue info (after the closure instruction)
	for _, upval := range inner.function.Upvalues {
		if upval.IsLocal {
			c.emitByte(1)
		} else {
			c.emitByte(0)
		}
		c.emitByte(upval.Index)
	}

	return nil
}

func (c *Compiler) compileThrow(e *ThrowExpr) error {
	// Compile the exception
	if err := c.compile(e.e); err != nil {
		return err
	}
	// For now, we'll panic - proper exception handling requires more work
	// This is a placeholder that will cause the VM to exit to AST evaluation
	return RT.NewError("throw not yet implemented in VM")
}

// Scope management

func (c *Compiler) beginScope() {
	c.scopeDepth++
}

func (c *Compiler) endScope() {
	c.scopeDepth--

	// Count locals to remove
	count := 0
	hasCaptured := false
	for c.localCount > 0 && c.locals[c.localCount-1].depth > c.scopeDepth {
		if c.locals[c.localCount-1].captured {
			hasCaptured = true
		}
		count++
		c.localCount--
	}

	// If any locals were captured, we need to close upvalues individually
	// For now, just use POPN for the simple case
	if hasCaptured {
		// More complex handling needed - close each upvalue then pop
		// For now, just pop all (upvalue closing is a TODO)
		if count > 0 {
			c.emitOp(OP_POPN)
			c.emitByte(byte(count))
		}
	} else if count > 0 {
		c.emitOp(OP_POPN)
		c.emitByte(byte(count))
	}
}

func (c *Compiler) addLocal(name Symbol) {
	if c.localCount >= 256 {
		panic(RT.NewError("Too many local variables in function"))
	}
	c.locals[c.localCount] = Local{
		name:  name,
		depth: c.scopeDepth,
	}
	c.localCount++
}

func (c *Compiler) resolveLocal(name Symbol) int {
	for i := c.localCount - 1; i >= 0; i-- {
		if c.locals[i].name.Equals(name) {
			return i
		}
	}
	return -1
}

func (c *Compiler) resolveUpvalue(name Symbol) int {
	if c.enclosing == nil {
		return -1
	}

	// Look for local in enclosing function
	local := c.enclosing.resolveLocal(name)
	if local >= 0 {
		c.enclosing.locals[local].captured = true
		return c.addUpvalue(uint8(local), true)
	}

	// Look for upvalue in enclosing function
	upvalue := c.enclosing.resolveUpvalue(name)
	if upvalue >= 0 {
		return c.addUpvalue(uint8(upvalue), false)
	}

	return -1
}

func (c *Compiler) addUpvalue(index uint8, isLocal bool) int {
	// Check if we already have this upvalue
	for i, uv := range c.function.Upvalues {
		if uv.Index == index && uv.IsLocal == isLocal {
			return i
		}
	}

	// Add new upvalue
	c.function.Upvalues = append(c.function.Upvalues, UpvalueInfo{
		Index:   index,
		IsLocal: isLocal,
	})
	return len(c.function.Upvalues) - 1
}

// Bytecode emission

func (c *Compiler) emitByte(b byte) {
	c.function.Chunk.AppendByte(b, 0) // TODO: proper line numbers
}

func (c *Compiler) emitOp(op Opcode) {
	c.function.Chunk.WriteOp(op, 0)
}

func (c *Compiler) emitShort(value uint16) {
	c.function.Chunk.WriteShort(value, 0)
}

func (c *Compiler) emitConstant(value Object) {
	idx := c.function.Chunk.AddConstant(value)
	if idx > 65535 {
		panic(RT.NewError("Too many constants in function"))
	}
	c.emitOp(OP_CONST)
	c.emitShort(uint16(idx))
}

func (c *Compiler) emitJump(op Opcode) int {
	c.emitOp(op)
	c.emitByte(0xff)
	c.emitByte(0xff)
	return len(c.function.Chunk.Code) - 2
}

func (c *Compiler) patchJump(offset int) {
	jump := len(c.function.Chunk.Code) - offset - 2
	if jump > 65535 {
		panic(RT.NewError("Jump too large"))
	}
	c.function.Chunk.Code[offset] = byte(jump >> 8)
	c.function.Chunk.Code[offset+1] = byte(jump & 0xff)
}

func (c *Compiler) emitReturn() {
	c.emitOp(OP_RETURN)
}

// IsVMCompatible checks if an expression can be compiled to bytecode.
func IsVMCompatible(expr Expr) bool {
	switch e := expr.(type) {
	case *LiteralExpr:
		return isLiteralVMCompatible(e.obj)
	case *VectorExpr:
		for _, elem := range e.v {
			if !IsVMCompatible(elem) {
				return false
			}
		}
		return true
	case *MapExpr:
		for i := range e.keys {
			if !IsVMCompatible(e.keys[i]) || !IsVMCompatible(e.values[i]) {
				return false
			}
		}
		return true
	case *SetExpr:
		for _, elem := range e.elements {
			if !IsVMCompatible(elem) {
				return false
			}
		}
		return true
	case *IfExpr:
		return IsVMCompatible(e.cond) && IsVMCompatible(e.positive) && IsVMCompatible(e.negative)
	case *DoExpr:
		for _, bodyExpr := range e.body {
			if !IsVMCompatible(bodyExpr) {
				return false
			}
		}
		return true
	case *LetExpr:
		for _, v := range e.values {
			if !IsVMCompatible(v) {
				return false
			}
		}
		for _, bodyExpr := range e.body {
			if !IsVMCompatible(bodyExpr) {
				return false
			}
		}
		return true
	case *LoopExpr:
		for _, v := range e.values {
			if !IsVMCompatible(v) {
				return false
			}
		}
		for _, bodyExpr := range e.body {
			if !IsVMCompatible(bodyExpr) {
				return false
			}
		}
		return true
	case *RecurExpr:
		for _, arg := range e.args {
			if !IsVMCompatible(arg) {
				return false
			}
		}
		return true
	case *VarRefExpr, *BindingExpr:
		return true
	case *CallExpr:
		if !IsVMCompatible(e.callable) {
			return false
		}
		for _, arg := range e.args {
			if !IsVMCompatible(arg) {
				return false
			}
		}
		return true
	case *DefExpr:
		return e.value == nil || IsVMCompatible(e.value)
	case *FnExpr:
		// Only support single-arity non-variadic functions for now
		if len(e.arities) != 1 || e.variadic != nil {
			return false
		}
		for _, bodyExpr := range e.arities[0].body {
			if !IsVMCompatible(bodyExpr) {
				return false
			}
		}
		return true
	case *MetaExpr:
		return IsVMCompatible(e.expr)
	case *TryExpr, *CatchExpr, *ThrowExpr, *MacroCallExpr:
		return false
	default:
		return false
	}
}

func isLiteralVMCompatible(obj Object) bool {
	switch obj.(type) {
	case Nil, Boolean, Int, Double, String, Char, Keyword, Symbol,
		*Ratio, *BigInt, *BigFloat, *Regex:
		return true
	default:
		return false
	}
}

// DisassembleChunk returns a string representation of the bytecode for debugging.
func DisassembleChunk(chunk *Chunk, name string) string {
	result := "== " + name + " ==\n"
	offset := 0
	for offset < len(chunk.Code) {
		result += disassembleInstruction(chunk, offset)
		offset = nextInstructionOffset(chunk, offset)
	}
	return result
}

func disassembleInstruction(chunk *Chunk, offset int) string {
	op := Opcode(chunk.Code[offset])
	result := padLeft(strconv.Itoa(offset), 4, "0") + " " + OpcodeName(op)

	switch op {
	case OP_CONST, OP_GET_VAR, OP_SET_VAR, OP_CLOSURE:
		idx := uint16(chunk.Code[offset+1])<<8 | uint16(chunk.Code[offset+2])
		if op == OP_CONST && int(idx) < len(chunk.Constants) {
			result += " " + strconv.Itoa(int(idx)) + " (" + chunk.Constants[idx].ToString(true) + ")"
		} else {
			result += " " + strconv.Itoa(int(idx))
		}
	case OP_GET_LOCAL, OP_SET_LOCAL, OP_GET_UPVALUE, OP_SET_UPVALUE, OP_CALL, OP_RECUR:
		result += " " + strconv.Itoa(int(chunk.Code[offset+1]))
	case OP_JUMP, OP_JUMP_IF_FALSE, OP_LOOP, OP_VECTOR, OP_MAP, OP_SET:
		val := uint16(chunk.Code[offset+1])<<8 | uint16(chunk.Code[offset+2])
		result += " " + strconv.Itoa(int(val))
	}

	return result + "\n"
}

func nextInstructionOffset(chunk *Chunk, offset int) int {
	op := Opcode(chunk.Code[offset])
	switch op {
	case OP_CONST, OP_GET_VAR, OP_SET_VAR, OP_JUMP, OP_JUMP_IF_FALSE, OP_LOOP, OP_VECTOR, OP_MAP, OP_SET:
		return offset + 3
	case OP_CLOSURE:
		// CLOSURE has variable length due to upvalue info
		idx := uint16(chunk.Code[offset+1])<<8 | uint16(chunk.Code[offset+2])
		// We'd need access to the parent's SubFunctions to get upvalue count
		// For now, return offset + 3 (the disassembler won't show upvalue info)
		_ = idx
		return offset + 3
	case OP_GET_LOCAL, OP_SET_LOCAL, OP_GET_UPVALUE, OP_SET_UPVALUE, OP_CALL, OP_RECUR:
		return offset + 2
	default:
		return offset + 1
	}
}

func padLeft(s string, length int, pad string) string {
	for len(s) < length {
		s = pad + s
	}
	return s
}

// IsVMCompatibleFn checks if a function expression can be compiled to bytecode.
func IsVMCompatibleFn(e *FnExpr) bool {
	// Only support single-arity non-variadic functions for now
	if len(e.arities) != 1 || e.variadic != nil {
		return false
	}
	for _, bodyExpr := range e.arities[0].body {
		if !IsVMCompatible(bodyExpr) {
			return false
		}
	}
	return true
}

// tryCompileFnExpr attempts to compile a FnExpr to bytecode.
func tryCompileFnExpr(e *FnExpr) (*FunctionProto, error) {
	name := "<anonymous>"
	if e.self.name != nil {
		name = e.self.Name()
	}
	return CompileFnArity(e.arities[0], name)
}
