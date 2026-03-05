package core

import (
	"strconv"
)

// Local represents a local variable in the compiler.
type Local struct {
	name     Symbol
	depth    int  // Scope depth
	captured bool // Whether captured by a closure
	slot     int  // Actual stack slot index (relative to frame.slots)
}

// Compiler compiles expressions to bytecode.
type Compiler struct {
	enclosing     *Compiler      // For nested functions
	function      *FunctionProto // Function being compiled
	locals        []Local        // Local variables in scope
	localCount    int            // Number of locals
	stackSize     int            // Current stack depth (locals + temporaries). Used to assign correct slot indices to new locals.
	scopeDepth    int            // Current scope depth
	loopStart     int            // Bytecode offset of loop start (for recur)
	loopDepth     int            // Scope depth at loop start
	loopSlotStart int            // Local slot index where loop variables start
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
	c.locals[0] = Local{depth: 0, slot: 0}
	c.localCount = 1
	c.stackSize = 1
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

// CompileTopLevel compiles a top-level expression to a zero-argument FunctionProto.
// This is used for file-level VM execution.
func CompileTopLevel(expr Expr) (*FunctionProto, error) {
	return Compile(expr, "<top-level>")
}

// CompileFnExpr compiles a function expression.
func CompileFnExpr(fnExpr *FnExpr, env *LocalEnv) (*FunctionProto, error) {
	name := "<anonymous>"
	if fnExpr.self.name != nil {
		name = fnExpr.self.Name()
	}

	proto := &FunctionProto{
		Name:    name,
		Arities: make([]*ArityProto, 0, len(fnExpr.arities)),
	}

	// Compile each fixed arity
	for _, arity := range fnExpr.arities {
		arityProto, _, err := compileArityProtoWithCompiler(arity, name, false)
		if err != nil {
			return nil, err
		}
		proto.Arities = append(proto.Arities, arityProto)
	}

	// Compile variadic arity
	if fnExpr.variadic != nil {
		varProto, _, err := compileArityProtoWithCompiler(*fnExpr.variadic, name, true)
		if err != nil {
			return nil, err
		}
		proto.VariadicArity = varProto
	}

	// Set legacy fields for single-arity case (backward compat)
	if len(proto.Arities) == 1 && proto.VariadicArity == nil {
		proto.Arity = proto.Arities[0].Arity
		proto.Chunk = proto.Arities[0].Chunk
		proto.Upvalues = proto.Arities[0].Upvalues
		proto.SubFunctions = proto.Arities[0].SubFunctions
	} else if len(proto.Arities) == 0 && proto.VariadicArity != nil {
		proto.Arity = proto.VariadicArity.Arity
		proto.Variadic = true
		proto.Chunk = proto.VariadicArity.Chunk
		proto.Upvalues = proto.VariadicArity.Upvalues
		proto.SubFunctions = proto.VariadicArity.SubFunctions
	}

	return proto, nil
}

// compileArityProtoWithCompiler compiles a single function arity and returns both the ArityProto and the compiler.
// The compiler is returned so callers can access SubFunctions if needed.
func compileArityProtoWithCompiler(arity FnArityExpr, name string, isVariadic bool) (*ArityProto, *Compiler, error) {
	c := NewCompiler(nil, name)

	// For variadic, Arity is fixed params count (excluding rest)
	fixedArgCount := len(arity.args)
	if isVariadic && fixedArgCount > 0 {
		fixedArgCount-- // Last arg is the rest param
	}

	// Add parameters as locals (including rest param for variadic)
	// Args are pushed by the VM before the body starts, so increment stackSize for each.
	for _, arg := range arity.args {
		c.stackSize++ // VM pushes this arg
		c.addLocal(arg)
	}

	// Compile the body
	for i, bodyExpr := range arity.body {
		if err := c.compile(bodyExpr); err != nil {
			return nil, nil, err
		}
		// Pop intermediate results except the last
		if i < len(arity.body)-1 {
			c.emitOp(OP_POP)
			c.stackSize--
		}
	}
	if len(arity.body) == 0 {
		c.emitOp(OP_NIL)
		c.stackSize++
	}

	c.emitReturn()

	ap := &ArityProto{
		Arity:        fixedArgCount,
		IsVariadic:   isVariadic,
		Chunk:        c.function.Chunk,
		Upvalues:     c.function.Upvalues,
		SubFunctions: c.function.SubFunctions,
	}
	extractArgTypes(arity, ap)
	return ap, c, nil
}

// CompileArityProto compiles a single function arity to an ArityProto.
func CompileArityProto(arity FnArityExpr, name string, isVariadic bool) (*ArityProto, error) {
	arityProto, _, err := compileArityProtoWithCompiler(arity, name, isVariadic)
	return arityProto, err
}

// CompileFnArity compiles a single function arity to bytecode (legacy, kept for compatibility).
func CompileFnArity(arity FnArityExpr, name string) (*FunctionProto, error) {
	arityProto, err := CompileArityProto(arity, name, false)
	if err != nil {
		return nil, err
	}
	return &FunctionProto{
		Name:     name,
		Arity:    arityProto.Arity,
		Chunk:    arityProto.Chunk,
		Upvalues: arityProto.Upvalues,
		Arities:  []*ArityProto{arityProto},
	}, nil
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
	case *SetMacroExpr:
		return c.compileSetMacro(e)
	case *MetaExpr:
		return c.compile(e.expr)
	case *ThrowExpr:
		return c.compileThrow(e)
	case *TryExpr:
		return c.compileTry(e)
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
	c.stackSize++ // net +1: pushed one value
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
	// OP_VECTOR pops N elements, pushes 1 vector. Each compile already added +1.
	c.stackSize += 1 - len(e.v)
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
	// OP_MAP pops 2N elements, pushes 1 map.
	c.stackSize += 1 - 2*len(e.keys)
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
	// OP_SET pops N elements, pushes 1 set.
	c.stackSize += 1 - len(e.elements)
	return nil
}

func (c *Compiler) compileIf(e *IfExpr) error {
	// Compile condition
	if err := c.compile(e.cond); err != nil {
		return err
	}

	// Jump to else branch if false (pops condition)
	elseJump := c.emitJump(OP_JUMP_IF_FALSE)
	c.stackSize-- // condition popped

	// Compile then branch
	baseStack := c.stackSize
	if err := c.compile(e.positive); err != nil {
		return err
	}

	// Jump over else branch
	endJump := c.emitJump(OP_JUMP)

	// Patch else jump - reset stack for else branch
	c.patchJump(elseJump)
	c.stackSize = baseStack

	// Compile else branch
	if err := c.compile(e.negative); err != nil {
		return err
	}

	// Patch end jump
	c.patchJump(endJump)
	// Both branches produce +1, so stackSize = baseStack + 1
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
			c.stackSize--
		}
	}
	// If body is empty, push nil
	if len(e.body) == 0 {
		c.emitOp(OP_NIL)
		c.stackSize++
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
			c.stackSize--
		}
	}
	if len(e.body) == 0 {
		c.emitOp(OP_NIL)
		c.stackSize++
	}

	c.endScope()
	return nil
}

func (c *Compiler) compileLoop(e *LoopExpr) error {
	c.beginScope()

	// Save previous loop state
	prevLoopStart := c.loopStart
	prevLoopDepth := c.loopDepth
	prevLoopSlotStart := c.loopSlotStart

	// Record where loop variables will be stored (before adding them)
	// Use stackSize since that's the actual stack position.
	c.loopSlotStart = c.stackSize

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
			c.stackSize--
		}
	}
	if len(e.body) == 0 {
		c.emitOp(OP_NIL)
		c.stackSize++
	}

	// Restore loop state
	c.loopStart = prevLoopStart
	c.loopDepth = prevLoopDepth
	c.loopSlotStart = prevLoopSlotStart

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

	// Emit recur instruction with arg count and slot start
	c.emitOp(OP_RECUR)
	c.emitByte(byte(len(e.args)))
	c.emitByte(byte(c.loopSlotStart))

	// OP_RECUR pops the new values and reassigns them to loop slots
	c.stackSize -= len(e.args)

	// Jump back to loop start
	offset := len(c.function.Chunk.Code) - c.loopStart + 3 // +3 for the LOOP instruction itself
	c.emitOp(OP_LOOP)
	c.emitShort(uint16(offset))

	// recur is always in tail position; code after is unreachable.
	// Push +1 to maintain the invariant that compile() nets +1.
	c.stackSize++

	return nil
}

func (c *Compiler) compileVarRef(e *VarRefExpr) error {
	idx := c.function.Chunk.AddConstant(e.vr)
	if idx > 65535 {
		panic(RT.NewError("Too many constants in function"))
	}
	c.emitOp(OP_GET_VAR)
	c.emitShort(uint16(idx))
	c.stackSize++
	return nil
}

func (c *Compiler) compileBinding(e *BindingExpr) error {
	// Find the local variable
	idx := c.resolveLocal(e.binding.name)
	if idx >= 0 {
		c.emitOp(OP_GET_LOCAL)
		c.emitByte(byte(idx))
		c.stackSize++
		return nil
	}

	// Check for upvalue
	idx = c.resolveUpvalue(e.binding.name)
	if idx >= 0 {
		c.emitOp(OP_GET_UPVALUE)
		c.emitByte(byte(idx))
		c.stackSize++
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
	// OP_CALL pops callable + N args, pushes 1 result. Net: -(N+1)+1 = -N.
	c.stackSize -= len(e.args)
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
	c.stackSize-- // pops 2, pushes 1
	return true, nil
}

func (c *Compiler) compileDef(e *DefExpr) error {
	// Only set the var's value if one is provided
	// (def x) leaves x unchanged, (def x val) sets x to val
	if e.value != nil {
		if err := c.compile(e.value); err != nil {
			return err
		}
		// Store in the var and pop the value
		idx := c.function.Chunk.AddConstant(e.vr)
		c.emitOp(OP_SET_VAR)
		c.emitShort(uint16(idx))
		c.emitOp(OP_POP) // Pop the value, leaving just the var on stack
		c.stackSize--     // compile pushed +1, POP removes it
	}

	// Set var metadata (matching AST evaluation behavior)
	// Build base metadata with line, column, file, ns, name
	baseMetaArr := EmptyArrayMap()
	baseMetaArr.Add(KEYWORDS.line, Int{I: e.startLine})
	baseMetaArr.Add(KEYWORDS.column, Int{I: e.startColumn})
	if e.filename != nil {
		baseMetaArr.Add(KEYWORDS.file, String{S: *e.filename})
	}
	baseMetaArr.Add(KEYWORDS.ns, e.vr.ns)
	baseMetaArr.Add(KEYWORDS.name, e.vr.name)
	var baseMeta Map = baseMetaArr
	if e.vr.isMacro {
		baseMeta = baseMeta.Assoc(KEYWORDS.macro, Boolean{B: true}).(Map)
	}

	varIdx := c.function.Chunk.AddConstant(e.vr)

	if e.meta != nil {
		// Compile user metadata and merge at runtime
		metaIdx := c.function.Chunk.AddConstant(baseMeta)
		c.emitOp(OP_SET_VAR_META)
		c.emitShort(uint16(varIdx))
		c.emitShort(uint16(metaIdx))

		if err := c.compile(e.meta); err != nil {
			return err
		}
		c.emitOp(OP_MERGE_VAR_META)
		c.emitShort(uint16(varIdx))
		c.stackSize-- // compile pushed +1, MERGE_VAR_META pops it
	} else {
		// Just set base metadata
		metaIdx := c.function.Chunk.AddConstant(baseMeta)
		c.emitOp(OP_SET_VAR_META)
		c.emitShort(uint16(varIdx))
		c.emitShort(uint16(metaIdx))
	}

	// Push the var as the result
	c.emitConstant(e.vr)
	c.stackSize++
	return nil
}

func (c *Compiler) compileSetMacro(e *SetMacroExpr) error {
	idx := c.function.Chunk.AddConstant(e.vr)
	if idx > 65535 {
		panic(RT.NewError("Too many constants in function"))
	}
	c.emitOp(OP_SET_MACRO)
	c.emitShort(uint16(idx))
	c.stackSize++ // VM pushes the var
	return nil
}

func (c *Compiler) compileFn(e *FnExpr) error {
	// Create a new compiler for the nested function
	name := "<anonymous>"
	if e.self.name != nil {
		name = e.self.Name()
	}

	// Compile a single arity using inner compiler
	compileInnerArity := func(arity FnArityExpr, isVariadic bool) (*ArityProto, *Compiler, error) {
		inner := NewCompiler(c, name)

		// If the function has a self-name, set slot 0 to it for self-recursion
		if e.self.name != nil {
			inner.locals[0].name = e.self
		}

		// For variadic, Arity is fixed params count (excluding rest)
		fixedArgCount := len(arity.args)
		if isVariadic && fixedArgCount > 0 {
			fixedArgCount-- // Last arg is the rest param
		}

		// Add parameters as locals (including rest param for variadic)
		// Args are pushed by the VM before the body starts, so increment stackSize for each.
		for _, arg := range arity.args {
			inner.stackSize++ // VM pushes this arg
			inner.addLocal(arg)
		}

		// Compile the body
		for i, bodyExpr := range arity.body {
			if err := inner.compile(bodyExpr); err != nil {
				return nil, nil, err
			}
			if i < len(arity.body)-1 {
				inner.emitOp(OP_POP)
				inner.stackSize--
			}
		}
		if len(arity.body) == 0 {
			inner.emitOp(OP_NIL)
			inner.stackSize++
		}

		inner.emitReturn()

		ap := &ArityProto{
			Arity:        fixedArgCount,
			IsVariadic:   isVariadic,
			Chunk:        inner.function.Chunk,
			Upvalues:     inner.function.Upvalues,
			SubFunctions: inner.function.SubFunctions,
		}
		extractArgTypes(arity, ap)
		return ap, inner, nil
	}

	// Build the function proto
	proto := &FunctionProto{
		Name:    name,
		Arities: make([]*ArityProto, 0, len(e.arities)),
	}

	// We need to track all upvalues from all arities
	// For simplicity, we use the first arity's compiler for upvalue emission
	var mainInner *Compiler

	// Compile each fixed arity
	for i, arity := range e.arities {
		arityProto, inner, err := compileInnerArity(arity, false)
		if err != nil {
			return err
		}
		proto.Arities = append(proto.Arities, arityProto)
		if i == 0 {
			mainInner = inner
		}
	}

	// Compile variadic arity
	if e.variadic != nil {
		varProto, inner, err := compileInnerArity(*e.variadic, true)
		if err != nil {
			return err
		}
		proto.VariadicArity = varProto
		if mainInner == nil {
			mainInner = inner
		}
	}

	// Set legacy fields - always set Upvalues from mainInner since that's what we emit
	if mainInner != nil {
		proto.Upvalues = mainInner.function.Upvalues
	}

	// Set other legacy fields for single-arity case (backward compat)
	if len(proto.Arities) == 1 && proto.VariadicArity == nil {
		proto.Arity = proto.Arities[0].Arity
		proto.Chunk = proto.Arities[0].Chunk
		proto.SubFunctions = proto.Arities[0].SubFunctions
	} else if len(proto.Arities) == 0 && proto.VariadicArity != nil {
		proto.Arity = proto.VariadicArity.Arity
		proto.Variadic = true
		proto.Chunk = proto.VariadicArity.Chunk
		proto.SubFunctions = proto.VariadicArity.SubFunctions
	}

	// Add the inner function to our sub-functions
	idx := c.function.AddSubFunction(proto)

	// Emit closure instruction
	c.emitOp(OP_CLOSURE)
	c.emitShort(uint16(idx))
	c.stackSize++ // pushes the closure

	// Emit upvalue info (after the closure instruction)
	if mainInner != nil {
		for _, upval := range mainInner.function.Upvalues {
			if upval.IsLocal {
				c.emitByte(1)
			} else {
				c.emitByte(0)
			}
			c.emitByte(upval.Index)
		}
	}

	return nil
}

func (c *Compiler) compileThrow(e *ThrowExpr) error {
	// Compile the exception
	if err := c.compile(e.e); err != nil {
		return err
	}
	c.emitOp(OP_THROW)
	// OP_THROW pops the exception and panics; code after is unreachable.
	// Net effect: compile pushed +1, throw pops -1 = 0. But we need +1 for invariant.
	// Since this is always in tail position, just leave stackSize as-is.
	return nil
}

func (c *Compiler) compileTry(e *TryExpr) error {
	// Create handler info - we'll fill in the IPs as we compile
	// Record stackSize so dispatchException can properly place the exception
	handler := HandlerInfo{
		Catches:       make([]CatchInfo, len(e.catches)),
		FinallyIP:     -1,
		EndIP:         -1,
		TryLocalCount: c.stackSize,
	}

	// Add placeholder handler and emit OP_TRY_BEGIN
	handlerIdx := c.function.Chunk.AddHandler(handler)
	c.emitOp(OP_TRY_BEGIN)
	c.emitShort(uint16(handlerIdx))

	// Compile try body
	baseStack := c.stackSize
	for i, bodyExpr := range e.body {
		if err := c.compile(bodyExpr); err != nil {
			return err
		}
		if i < len(e.body)-1 {
			c.emitOp(OP_POP)
			c.stackSize--
		}
	}
	if len(e.body) == 0 {
		c.emitOp(OP_NIL)
		c.stackSize++
	}

	// Normal exit from try - pop handler and jump to after finally
	c.emitOp(OP_TRY_END)
	exitJump := c.emitJump(OP_JUMP)

	// Compile each catch clause
	for i, catch := range e.catches {
		catchIP := len(c.function.Chunk.Code)

		// Reset stack to base for catch entry (try body's stack is unwound)
		c.stackSize = baseStack

		// Begin scope for catch binding
		c.beginScope()

		// The exception is pushed by the VM's exception handler
		c.stackSize++ // VM pushes exception
		c.addLocal(catch.excSymbol)

		// Record catch info
		c.function.Chunk.Handlers[handlerIdx].Catches[i] = CatchInfo{
			ExcType:   catch.excType,
			HandlerIP: catchIP,
			LocalSlot: c.locals[c.localCount-1].slot,
		}

		// Compile catch body
		for j, bodyExpr := range catch.body {
			if err := c.compile(bodyExpr); err != nil {
				return err
			}
			if j < len(catch.body)-1 {
				c.emitOp(OP_POP)
				c.stackSize--
			}
		}
		if len(catch.body) == 0 {
			c.emitOp(OP_NIL)
			c.stackSize++
		}

		// Custom scope cleanup for catch - use OP_POP_SLOT to remove the exception
		// binding at its specific slot, preserving any operands above it
		c.scopeDepth--
		catchSlot := c.function.Chunk.Handlers[handlerIdx].TryLocalCount
		if c.localCount > 0 && c.locals[c.localCount-1].depth > c.scopeDepth {
			c.emitOp(OP_POP_SLOT)
			c.emitByte(byte(catchSlot))
			c.localCount--
			c.stackSize--
		}

		// Jump to after finally (unless this is the last catch and there's no finally)
		if i < len(e.catches)-1 || e.finallyExpr != nil {
			c.patchJump(exitJump)
			exitJump = c.emitJump(OP_JUMP)
		}
	}

	// Compile finally block if present
	if e.finallyExpr != nil {
		c.function.Chunk.Handlers[handlerIdx].FinallyIP = len(c.function.Chunk.Code)

		// Finally body - result is discarded
		for _, bodyExpr := range e.finallyExpr {
			if err := c.compile(bodyExpr); err != nil {
				return err
			}
			c.emitOp(OP_POP)
			c.stackSize--
		}
	}

	// Patch the exit jump to come here
	c.patchJump(exitJump)
	c.function.Chunk.Handlers[handlerIdx].EndIP = len(c.function.Chunk.Code)

	// Net effect: +1 (try body result or catch body result)
	return nil
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
			c.stackSize -= count
		}
	} else if count > 0 {
		c.emitOp(OP_POPN)
		c.emitByte(byte(count))
		c.stackSize -= count
	}
}

func (c *Compiler) addLocal(name Symbol) {
	if c.localCount >= 256 {
		panic(RT.NewError("Too many local variables in function"))
	}
	// The value for this local was just pushed to the stack at position stackSize-1.
	// But we don't increment stackSize here because it was already incremented
	// when the value was pushed (by the code that compiled the init expression).
	// We just record where it is.
	c.locals[c.localCount] = Local{
		name:  name,
		depth: c.scopeDepth,
		slot:  c.stackSize - 1,
	}
	c.localCount++
}

func (c *Compiler) resolveLocal(name Symbol) int {
	for i := c.localCount - 1; i >= 0; i-- {
		if c.locals[i].name.Equals(name) {
			return c.locals[i].slot
		}
	}
	return -1
}

// resolveLocalFull returns (slot, arrayIndex) for the named local, or (-1, -1).
func (c *Compiler) resolveLocalFull(name Symbol) (int, int) {
	for i := c.localCount - 1; i >= 0; i-- {
		if c.locals[i].name.Equals(name) {
			return c.locals[i].slot, i
		}
	}
	return -1, -1
}

func (c *Compiler) resolveUpvalue(name Symbol) int {
	if c.enclosing == nil {
		return -1
	}

	// Look for local in enclosing function
	slot, idx := c.enclosing.resolveLocalFull(name)
	if slot >= 0 {
		c.enclosing.locals[idx].captured = true
		return c.addUpvalue(uint8(slot), true)
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
	case *VarRefExpr, *BindingExpr, *SetMacroExpr:
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
		if e.meta != nil && !IsVMCompatible(e.meta) {
			return false
		}
		return e.value == nil || IsVMCompatible(e.value)
	case *FnExpr:
		// Check all arities
		for _, arity := range e.arities {
			for _, bodyExpr := range arity.body {
				if !IsVMCompatible(bodyExpr) {
					return false
				}
			}
		}
		if e.variadic != nil {
			for _, bodyExpr := range e.variadic.body {
				if !IsVMCompatible(bodyExpr) {
					return false
				}
			}
		}
		return true
	case *MetaExpr:
		return IsVMCompatible(e.expr)
	case *ThrowExpr:
		return IsVMCompatible(e.e)
	case *TryExpr:
		for _, bodyExpr := range e.body {
			if !IsVMCompatible(bodyExpr) {
				return false
			}
		}
		for _, catch := range e.catches {
			for _, bodyExpr := range catch.body {
				if !IsVMCompatible(bodyExpr) {
					return false
				}
			}
		}
		if e.finallyExpr != nil {
			for _, bodyExpr := range e.finallyExpr {
				if !IsVMCompatible(bodyExpr) {
					return false
				}
			}
		}
		return true
	case *CatchExpr, *MacroCallExpr:
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

// extractArgTypes extracts type tag info from FnArityExpr args and stores them in the ArityProto.
func extractArgTypes(arity FnArityExpr, ap *ArityProto) {
	// Always extract return type tag from arity
	ap.TaggedType = arity.taggedType

	// Check if any args have type tags
	hasTypes := false
	for _, arg := range arity.args {
		if m := arg.GetMeta(); m != nil {
			if ok, _ := m.Get(KEYWORDS.tag); ok {
				hasTypes = true
				break
			}
		}
	}
	if !hasTypes {
		return
	}

	ap.ArgTypes = make([][]*Type, len(arity.args))
	for i, arg := range arity.args {
		if m := arg.GetMeta(); m != nil {
			if ok, typeName := m.Get(KEYWORDS.tag); ok {
				switch typeDecl := typeName.(type) {
				case Symbol:
					if t := TYPES[typeDecl.name]; t != nil {
						ap.ArgTypes[i] = []*Type{t}
					}
				case String:
					parts := splitString(typeDecl.S, '|')
					for _, p := range parts {
						if t := TYPES[MakeSymbol(p).name]; t != nil {
							ap.ArgTypes[i] = append(ap.ArgTypes[i], t)
						}
					}
				}
			}
		}
	}
}

// splitString splits a string by a separator character.
func splitString(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
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
	case OP_CONST, OP_GET_VAR, OP_SET_VAR, OP_CLOSURE, OP_SET_MACRO:
		idx := uint16(chunk.Code[offset+1])<<8 | uint16(chunk.Code[offset+2])
		if op == OP_CONST && int(idx) < len(chunk.Constants) {
			result += " " + strconv.Itoa(int(idx)) + " (" + chunk.Constants[idx].ToString(true) + ")"
		} else {
			result += " " + strconv.Itoa(int(idx))
		}
	case OP_GET_LOCAL, OP_SET_LOCAL, OP_GET_UPVALUE, OP_SET_UPVALUE, OP_CALL:
		result += " " + strconv.Itoa(int(chunk.Code[offset+1]))
	case OP_RECUR:
		result += " argCount=" + strconv.Itoa(int(chunk.Code[offset+1])) + " slotStart=" + strconv.Itoa(int(chunk.Code[offset+2]))
	case OP_JUMP, OP_JUMP_IF_FALSE, OP_LOOP, OP_VECTOR, OP_MAP, OP_SET, OP_TRY_BEGIN:
		val := uint16(chunk.Code[offset+1])<<8 | uint16(chunk.Code[offset+2])
		result += " " + strconv.Itoa(int(val))
	}

	return result + "\n"
}

func nextInstructionOffset(chunk *Chunk, offset int) int {
	op := Opcode(chunk.Code[offset])
	switch op {
	case OP_CONST, OP_GET_VAR, OP_SET_VAR, OP_JUMP, OP_JUMP_IF_FALSE, OP_LOOP, OP_VECTOR, OP_MAP, OP_SET, OP_TRY_BEGIN, OP_SET_MACRO:
		return offset + 3
	case OP_CLOSURE:
		// CLOSURE has variable length due to upvalue info
		idx := uint16(chunk.Code[offset+1])<<8 | uint16(chunk.Code[offset+2])
		// We'd need access to the parent's SubFunctions to get upvalue count
		// For now, return offset + 3 (the disassembler won't show upvalue info)
		_ = idx
		return offset + 3
	case OP_GET_LOCAL, OP_SET_LOCAL, OP_GET_UPVALUE, OP_SET_UPVALUE, OP_CALL:
		return offset + 2
	case OP_RECUR:
		return offset + 3
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
	// Check all arities
	for _, arity := range e.arities {
		for _, bodyExpr := range arity.body {
			if !IsVMCompatible(bodyExpr) {
				return false
			}
		}
	}
	if e.variadic != nil {
		for _, bodyExpr := range e.variadic.body {
			if !IsVMCompatible(bodyExpr) {
				return false
			}
		}
	}
	return true
}

// tryCompileFnExpr attempts to compile a FnExpr to bytecode.
func tryCompileFnExpr(e *FnExpr) (*FunctionProto, error) {
	return CompileFnExpr(e, nil)
}
