package core

import (
	"fmt"
	"math"
	"strings"
)

// A compiler local is identified by the parser's lexical binding, never its spelling.
type Local struct {
	binding *Binding
	depth   int
	slot    int
}

type Compiler struct {
	enclosing     *Compiler
	function      *FunctionProto
	captures      *[]UpvalueInfo // shared by every arity of a function
	locals        []Local
	stackSize     int
	scopeDepth    int
	loopStart     int
	loopSlotStart int
	currentPos    Position
	closedEnv     *LocalEnv
}

func NewCompiler(enclosing *Compiler, name string) *Compiler {
	c := &Compiler{enclosing: enclosing, function: NewFunctionProto(name), stackSize: 1, loopStart: -1}
	c.captures = &c.function.Upvalues
	return c
}

func Compile(expr Expr, name string) (*FunctionProto, error) {
	c := NewCompiler(nil, name)
	if err := c.compile(expr); err != nil {
		return nil, err
	}
	c.currentPos = expr.Pos()
	c.emitOp(OP_RETURN)
	return c.function, nil
}

func CompileTopLevel(expr Expr) (*FunctionProto, error) { return Compile(expr, "<top-level>") }

func CompileFnExpr(expr *FnExpr, env *LocalEnv) (*FunctionProto, error) {
	return compileFunction(expr, nil, env)
}

// All arities share one capture layout. Nested functions refer to that same layout.
func compileFunction(expr *FnExpr, parent *Compiler, env *LocalEnv) (*FunctionProto, error) {
	name := "<anonymous>"
	if expr.self.name != nil {
		name = expr.self.Name()
	}
	proto := &FunctionProto{Name: name}
	compileArity := func(a FnArityExpr, variadic bool) (*ArityProto, error) {
		c := NewCompiler(parent, name)
		c.captures = &proto.Upvalues
		c.closedEnv = env
		if expr.selfBinding != nil {
			c.locals = append(c.locals, Local{binding: expr.selfBinding, slot: 0})
		}
		for i := range a.args {
			c.stackSize++
			c.addLocal(a.bindings[i])
		}
		c.loopStart, c.loopSlotStart = 0, 1
		if err := c.compileBody(a.body); err != nil {
			return nil, err
		}
		c.currentPos = a.Position
		c.emitOp(OP_RETURN)
		fixed := len(a.args)
		if variadic {
			fixed--
		}
		ap := &ArityProto{Arity: fixed, IsVariadic: variadic, Chunk: c.function.Chunk, SubFunctions: c.function.SubFunctions}
		extractArgTypes(a, ap)
		return ap, nil
	}
	for _, a := range expr.arities {
		ap, err := compileArity(a, false)
		if err != nil {
			return nil, err
		}
		proto.Arities = append(proto.Arities, ap)
	}
	if expr.variadic != nil {
		ap, err := compileArity(*expr.variadic, true)
		if err != nil {
			return nil, err
		}
		proto.VariadicArity = ap
	}
	return proto, nil
}

func CompileArityProto(a FnArityExpr, name string, variadic bool) (*ArityProto, error) {
	e := &FnExpr{}
	if variadic {
		e.variadic = &a
	} else {
		e.arities = []FnArityExpr{a}
	}
	p, err := compileFunction(e, nil, nil)
	if err != nil {
		return nil, err
	}
	if variadic {
		return p.VariadicArity, nil
	}
	return p.Arities[0], nil
}

func CompileFnArity(a FnArityExpr, name string) (*FunctionProto, error) {
	return compileFunction(&FnExpr{arities: []FnArityExpr{a}}, nil, nil)
}

func (c *Compiler) compile(expr Expr) error {
	previous := c.currentPos
	c.currentPos = expr.Pos()
	defer func() { c.currentPos = previous }()
	switch e := expr.(type) {
	case *LiteralExpr:
		c.emitConstant(e.obj)
		c.stackSize++
	case *VectorExpr:
		for _, v := range e.v {
			if err := c.compile(v); err != nil {
				return err
			}
		}
		c.emitOp(OP_VECTOR)
		c.emitOperand(len(e.v))
		c.stackSize += 1 - len(e.v)
	case *MapExpr:
		c.emitOp(OP_MAP_NEW)
		c.emitOperand(len(e.keys))
		c.stackSize++
		for i := range e.keys {
			if err := c.compile(e.keys[i]); err != nil {
				return err
			}
			c.emitOp(OP_MAP_CHECK)
			if err := c.compile(e.values[i]); err != nil {
				return err
			}
			c.emitOp(OP_MAP_ADD)
			c.stackSize -= 2
		}
	case *SetExpr:
		c.emitOp(OP_SET_NEW)
		c.stackSize++
		for _, v := range e.elements {
			if err := c.compile(v); err != nil {
				return err
			}
			c.emitOp(OP_SET_ADD)
			c.stackSize--
		}
	case *MetaExpr:
		if err := c.compile(e.meta); err != nil {
			return err
		}
		if err := c.compile(e.expr); err != nil {
			return err
		}
		c.emitOp(OP_WITH_META)
		c.stackSize--
	case *IfExpr:
		if err := c.compile(e.cond); err != nil {
			return err
		}
		otherwise := c.emitJump(OP_JUMP_IF_FALSE)
		c.stackSize--
		base := c.stackSize
		if err := c.compile(e.positive); err != nil {
			return err
		}
		end := c.emitJump(OP_JUMP)
		c.patchJump(otherwise)
		c.stackSize = base
		if err := c.compile(e.negative); err != nil {
			return err
		}
		c.patchJump(end)
	case *DoExpr:
		return c.compileBody(e.body)
	case *LetExpr:
		return c.compileLet(e, false)
	case *LoopExpr:
		return c.compileLet((*LetExpr)(e), !e.recursive)
	case *RecurExpr:
		if c.loopStart < 0 {
			return RT.NewError("recur outside of loop")
		}
		for _, a := range e.args {
			if err := c.compile(a); err != nil {
				return err
			}
		}
		c.emitOp(OP_RECUR)
		c.emitOperand(len(e.args))
		c.emitOperand(c.loopSlotStart)
		c.emitOp(OP_LOOP)
		c.emitOperand(len(c.function.Chunk.Code) + 4 - c.loopStart)
		c.stackSize += 1 - len(e.args) // unreachable continuation still has expression stack shape
	case *BindingExpr:
		if slot := c.resolveLocal(e.binding); slot >= 0 {
			c.emitOp(OP_GET_LOCAL)
			c.emitOperand(slot)
		} else if slot := c.resolveUpvalue(e.binding); slot >= 0 {
			c.emitOp(OP_GET_UPVALUE)
			c.emitOperand(slot)
		} else {
			root := c
			for root.enclosing != nil {
				root = root.enclosing
			}
			env := root.closedEnv
			for env != nil && env.frame > e.binding.frame {
				env = env.parent
			}
			if env == nil || env.frame != e.binding.frame || e.binding.index >= len(env.bindings) || env.bindings[e.binding.index] == nil {
				return RT.NewErrorWithPos("Cannot resolve binding: "+e.binding.name.ToString(false), e.Pos())
			}
			c.emitConstant(env.bindings[e.binding.index])
		}
		c.stackSize++
	case *VarRefExpr:
		c.emitOp(OP_GET_VAR)
		c.emitOperand(c.function.Chunk.AddConstant(e.vr))
		c.stackSize++
	case *CallExpr:
		// Vars are mutable. Do not substitute name-based arithmetic intrinsics.
		if err := c.compile(e.callable); err != nil {
			return err
		}
		previous := c.currentPos
		c.currentPos = e.callable.Pos()
		c.emitOp(OP_CHECK_CALLABLE)
		c.currentPos = previous
		for _, a := range e.args {
			if err := c.compile(a); err != nil {
				return err
			}
		}
		c.emitCall(len(e.args), e.Name())
		c.stackSize -= len(e.args)
	case *FnExpr:
		proto, err := compileFunction(e, c, nil)
		if err != nil {
			return err
		}
		c.emitOp(OP_CLOSURE)
		c.emitOperand(c.function.AddSubFunction(proto))
		c.stackSize++
	case *DefExpr:
		idx := c.function.Chunk.AddConstant(e.vr)
		if e.value != nil {
			if err := c.compile(e.value); err != nil {
				return err
			}
			c.emitOp(OP_SET_VAR)
			c.emitOperand(idx)
			c.emitOp(OP_POP)
			c.stackSize--
		}
		m := EmptyArrayMap()
		m.Add(KEYWORDS.line, Int{I: e.startLine})
		m.Add(KEYWORDS.column, Int{I: e.startColumn})
		m.Add(KEYWORDS.file, String{S: e.Position.Filename()})
		m.Add(KEYWORDS.ns, e.vr.ns)
		m.Add(KEYWORDS.name, e.vr.name)
		c.emitOp(OP_SET_VAR_META)
		c.emitOperand(idx)
		c.emitOperand(c.function.Chunk.AddConstant(m))
		if e.meta != nil {
			if err := c.compile(e.meta); err != nil {
				return err
			}
			c.emitOp(OP_MERGE_VAR_META)
			c.emitOperand(idx)
			c.stackSize--
		}
		// isMacro may change after compilation; apply it after user metadata, as AST does.
		c.emitOp(OP_FINISH_DEF)
		c.emitOperand(idx)
		c.stackSize++
	case *SetMacroExpr:
		c.emitOp(OP_SET_MACRO)
		c.emitOperand(c.function.Chunk.AddConstant(e.vr))
		c.stackSize++
	case *ThrowExpr:
		if err := c.compile(e.e); err != nil {
			return err
		}
		c.emitOp(OP_THROW)
	case *TryExpr:
		return c.compileTry(e)
	case *MacroCallExpr:
		c.emitConstant(e.macro.(Object))
		c.stackSize++
		for _, a := range e.args {
			c.emitConstant(a)
			c.stackSize++
		}
		c.emitCall(len(e.args), e.Name())
		c.stackSize -= len(e.args)
	default:
		return RT.NewErrorWithPos(fmt.Sprintf("Invalid runtime expression %T", expr), expr.Pos())
	}
	return nil
}

func (c *Compiler) compileBody(body []Expr) error {
	for i, e := range body {
		if err := c.compile(e); err != nil {
			return err
		}
		if i+1 < len(body) {
			c.emitOp(OP_POP)
			c.stackSize--
		}
	}
	if len(body) == 0 {
		c.emitOp(OP_NIL)
		c.stackSize++
	}
	return nil
}

func (c *Compiler) compileLet(e *LetExpr, loop bool) error {
	c.scopeDepth++
	if e.recursive {
		// Heap cells are shared only for recursive initialization. Ordinary lexical
		// captures are snapshots, including closures created before a recur.
		for i := range e.names {
			c.emitOp(OP_NIL)
			c.emitOp(OP_BIND_CELL)
			c.stackSize++
			c.addLocal(e.bindings[i])
		}
	}
	for i, v := range e.values {
		if err := c.compile(v); err != nil {
			return err
		}
		if e.recursive {
			c.emitOp(OP_SET_LOCAL)
			c.emitOperand(c.resolveLocal(e.bindings[i]))
			c.emitOp(OP_POP)
			c.stackSize--
		} else {
			c.addLocal(e.bindings[i])
		}
	}
	prevStart, prevSlot := c.loopStart, c.loopSlotStart
	if loop {
		c.loopStart = len(c.function.Chunk.Code)
		c.loopSlotStart = c.stackSize - len(e.names)
	}
	if err := c.compileBody(e.body); err != nil {
		return err
	}
	c.loopStart, c.loopSlotStart = prevStart, prevSlot
	c.endScope()
	return nil
}

func (c *Compiler) compileTry(e *TryExpr) error {
	handler := HandlerInfo{Catches: make([]CatchInfo, len(e.catches)), FinallyIP: -1, EndIP: -1, TryLocalCount: c.stackSize}
	idx := c.function.Chunk.AddHandler(handler)
	c.emitOp(OP_TRY_BEGIN)
	c.emitOperand(idx)
	base := c.stackSize
	if err := c.compileBody(e.body); err != nil {
		return err
	}
	c.emitOp(OP_TRY_END)
	exits := []int{c.emitJump(OP_JUMP)}
	var guards []int
	for i, catch := range e.catches {
		ip := len(c.function.Chunk.Code)
		c.stackSize = base + 1
		c.scopeDepth++
		c.addLocal(catch.binding)
		c.function.Chunk.Handlers[idx].Catches[i] = CatchInfo{ExcType: catch.excType, HandlerIP: ip, LocalSlot: base}
		if e.finallyExpr != nil {
			guard := c.function.Chunk.AddHandler(HandlerInfo{FinallyIP: -1, EndIP: -1, TryLocalCount: base})
			guards = append(guards, guard)
			c.emitOp(OP_TRY_BEGIN)
			c.emitOperand(guard)
		}
		if err := c.compileBody(catch.body); err != nil {
			return err
		}
		if e.finallyExpr != nil {
			c.emitOp(OP_TRY_END)
		}
		c.endScope()
		exits = append(exits, c.emitJump(OP_JUMP))
	}
	finallyIP := len(c.function.Chunk.Code)
	for _, jump := range exits {
		c.patchJump(jump)
	}
	c.stackSize = base + 1
	if e.finallyExpr != nil {
		c.function.Chunk.Handlers[idx].FinallyIP = finallyIP
		for _, guard := range guards {
			c.function.Chunk.Handlers[guard].FinallyIP = finallyIP
		}
		for _, b := range e.finallyExpr {
			if err := c.compile(b); err != nil {
				return err
			}
			c.emitOp(OP_POP)
			c.stackSize--
		}
		c.emitOp(OP_FINALLY_END)
	}
	end := len(c.function.Chunk.Code)
	c.function.Chunk.Handlers[idx].EndIP = end
	for _, guard := range guards {
		c.function.Chunk.Handlers[guard].EndIP = end
	}
	return nil
}

func (c *Compiler) addLocal(b *Binding) {
	c.locals = append(c.locals, Local{binding: b, depth: c.scopeDepth, slot: c.stackSize - 1})
}

func (c *Compiler) endScope() {
	c.scopeDepth--
	count := 0
	for len(c.locals) > 0 && c.locals[len(c.locals)-1].depth > c.scopeDepth {
		c.locals = c.locals[:len(c.locals)-1]
		count++
	}
	if count > 0 {
		c.emitOp(OP_POPN)
		c.emitOperand(count)
		c.stackSize -= count
	}
}

func (c *Compiler) resolveLocal(b *Binding) int {
	for i := len(c.locals) - 1; i >= 0; i-- {
		if c.locals[i].binding == b {
			return c.locals[i].slot
		}
	}
	return -1
}

func (c *Compiler) resolveUpvalue(b *Binding) int {
	if c.enclosing == nil {
		return -1
	}
	slot := c.enclosing.resolveLocal(b)
	local := slot >= 0
	if !local {
		slot = c.enclosing.resolveUpvalue(b)
	}
	if slot < 0 {
		return -1
	}
	for i, u := range *c.captures {
		if u.Index == slot && u.IsLocal == local {
			return i
		}
	}
	*c.captures = append(*c.captures, UpvalueInfo{Index: slot, IsLocal: local})
	return len(*c.captures) - 1
}

func (c *Compiler) emitCall(argc int, name string) {
	ip := len(c.function.Chunk.Code)
	c.emitOp(OP_CALL)
	if site := c.function.Chunk.callSites[ip]; site != nil {
		site.callName = name
	}
	c.emitOperand(argc)
}

func (c *Compiler) emitOp(op Opcode) {
	chunk := c.function.Chunk
	if op == OP_CALL {
		if chunk.callSites == nil {
			chunk.callSites = make(map[int]*CallExpr)
		}
		chunk.callSites[len(chunk.Code)] = &CallExpr{Position: c.currentPos}
	}
	chunk.appendAt(byte(op), c.currentPos)
}

// Every numeric instruction operand uses an unsigned 32-bit encoding. Reject
// overflow before narrowing; temporaries count toward local slot indices too.
func (c *Compiler) emitOperand(n int) {
	if n < 0 || uint64(n) > math.MaxUint32 {
		panic(RT.NewErrorWithPos("Bytecode operand overflow", c.currentPos))
	}
	for shift := 24; shift >= 0; shift -= 8 {
		c.function.Chunk.appendAt(byte(uint32(n)>>shift), c.currentPos)
	}
}
func (c *Compiler) emitConstant(v Object) {
	c.emitOp(OP_CONST)
	c.emitOperand(c.function.Chunk.AddConstant(v))
}
func (c *Compiler) emitJump(op Opcode) int {
	c.emitOp(op)
	offset := len(c.function.Chunk.Code)
	c.emitOperand(0)
	return offset
}
func (c *Compiler) patchJump(offset int) {
	n := len(c.function.Chunk.Code) - offset - 4
	if uint64(n) > math.MaxUint32 {
		panic(RT.NewError("Jump too large"))
	}
	for i := 0; i < 4; i++ {
		c.function.Chunk.Code[offset+i] = byte(uint32(n) >> uint(24-8*i))
	}
}

// Compatibility queries are retained for callers/tests, not as runtime admission
// gates. Runtime constants do not have to be serializable.
func IsVMCompatible(e Expr) bool      { _, err := CompileTopLevel(e); return err == nil }
func IsVMCompatibleFn(e *FnExpr) bool { _, err := CompileFnExpr(e, nil); return err == nil }

func extractArgTypes(a FnArityExpr, ap *ArityProto) {
	if len(a.taggedTypes) > 0 {
		ap.TaggedType = a.taggedTypes[0]
	}
	ap.ArgTypes = make([][]*Type, len(a.args))
	for i, arg := range a.args {
		if m := arg.GetMeta(); m != nil {
			if ok, tag := m.Get(KEYWORDS.tag); ok {
				var names []string
				switch v := tag.(type) {
				case Symbol:
					names = []string{v.Name()}
				case String:
					names = strings.Split(v.S, "|")
				}
				for _, name := range names {
					if t := TYPES[MakeSymbol(name).name]; t != nil {
						ap.ArgTypes[i] = append(ap.ArgTypes[i], t)
					}
				}
			}
		}
	}
}

func DisassembleChunk(chunk *Chunk, name string) string {
	var out strings.Builder
	fmt.Fprintf(&out, "== %s ==\n", name)
	for offset := 0; offset < len(chunk.Code); {
		op := Opcode(chunk.Code[offset])
		fmt.Fprintf(&out, "%04d %s", offset, OpcodeName(op))
		count, ok := opcodeOperands(op)
		if !ok || offset+1+count*4 > len(chunk.Code) {
			out.WriteString(" <invalid>\n")
			break
		}
		for i := 0; i < count; i++ {
			fmt.Fprintf(&out, " %d", operandAt(chunk.Code, offset+1+4*i))
		}
		out.WriteByte('\n')
		offset += 1 + count*4
	}
	return out.String()
}
