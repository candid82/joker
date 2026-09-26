package core

import (
	"strconv"
	"strings"
	"sync"
)

// Captures never point into VM storage. Ordinary bindings are snapshots;
// letfn initialization shares heap cells between the mutually recursive closures.
type Upvalue struct{ closed Object }
type bindingCell struct{ Object }

func capturedValue(v Object) Object {
	if cell, ok := v.(*bindingCell); ok {
		return cell.Object
	}
	return v
}

type CallFrame struct {
	closure    *Fn
	arityProto *ArityProto
	ip         int
	lastOp     int
	slots      int
}
type ExceptionHandler struct {
	handlerIdx int
	frameIndex int
	stackTop   int
}
type pendingFinally struct {
	err          interface{}
	frameIndex   int
	finishIP     int
	handlerDepth int
}

// Context handles are invalidated before a VM returns to its pool. Native code
// may release the GIL and interleave executions, so a saved runtime context must
// never point to a pooled VM that has subsequently been reused.
type vmContext struct {
	vm     *VM
	parent *vmContext
	entry  Expr
}

type VM struct {
	context        *vmContext
	stack          []Object
	stackTop       int
	stackHighWater int
	frames         []CallFrame
	frameCount     int
	handlers       []ExceptionHandler
	handlerCount   int
	pending        []pendingFinally
	pendingCount   int
}

func NewVM() *VM {
	return &VM{stack: make([]Object, 256), frames: make([]CallFrame, 64), handlers: make([]ExceptionHandler, 16), pending: make([]pendingFinally, 16)}
}
func (vm *VM) Reset() {
	clear(vm.stack[:vm.stackHighWater])
	clear(vm.frames[:vm.frameCount])
	clear(vm.handlers[:vm.handlerCount])
	clear(vm.pending[:vm.pendingCount])
	vm.stackHighWater = 0
	vm.stackTop, vm.frameCount, vm.handlerCount, vm.pendingCount = 0, 0, 0, 0
	vm.context = nil
}
func (vm *VM) ensureStack(n int) {
	vm.stackHighWater = max(vm.stackHighWater, n)
	if n > len(vm.stack) {
		vm.stack = append(vm.stack, make([]Object, max(n-len(vm.stack), len(vm.stack)))...)
	}
}
func (vm *VM) Push(v Object) {
	if v == nil {
		panic(RT.NewError("VM invariant: nil value"))
	}
	vm.ensureStack(vm.stackTop + 1)
	vm.stack[vm.stackTop] = v
	vm.stackTop++
}
func (vm *VM) Pop() Object {
	if vm.stackTop == 0 {
		panic(RT.NewError("VM stack underflow"))
	}
	vm.stackTop--
	v := vm.stack[vm.stackTop]
	vm.stack[vm.stackTop] = nil
	if v == nil {
		panic(RT.NewError("VM invariant: nil stack slot"))
	}
	return v
}
func (vm *VM) Peek(distance int) Object { return vm.stack[vm.stackTop-1-distance] }
func (vm *VM) PopN(n int) []Object {
	args := make([]Object, n)
	for i := n - 1; i >= 0; i-- {
		args[i] = vm.Pop()
	}
	return args
}
func (vm *VM) truncate(n int) {
	if n < vm.stackTop {
		clear(vm.stack[n:vm.stackTop])
	}
	vm.stackTop = n
}
func (vm *VM) Execute(fn *Fn, args []Object) Object {
	vm.Reset()
	previous := RT.vm
	context := &vmContext{vm: vm, parent: previous, entry: RT.currentExpr}
	vm.context = context
	RT.vm = context
	defer func() { context.vm = nil; context.parent = nil; context.entry = nil; RT.vm = previous }()
	vm.Push(fn)
	for _, arg := range args {
		vm.Push(arg)
	}
	vm.callFn(fn, len(args))
	return vm.run()
}
func (vm *VM) ExecuteTopLevel(proto *FunctionProto) Object {
	return vm.Execute(&Fn{proto: proto, isCompiled: true}, nil)
}

// A single recovery boundary covers a stretch of bytecode. Normal returns do
// not panic. Host panics unwind finally too, but never match Joker catches.
func (vm *VM) run() Object {
	frame := &vm.frames[vm.frameCount-1]
	chunk := frame.arityProto.Chunk
	for {
		var failure interface{}
		result := func() Object {
			defer func() { failure = recover() }()
			for {
				if result := vm.executeOneOp(&frame, &chunk); result != nil {
					return result
				}
			}
		}()
		if failure == nil {
			return result
		}
		if vm.dispatchException(failure, &frame, &chunk) {
			continue
		}
		panic(failure)
	}
}
func (vm *VM) readOperand(f *CallFrame, c *Chunk) int {
	n := operandAt(c.Code, f.ip)
	f.ip += 4
	return n
}
func (vm *VM) executeOneOp(fp **CallFrame, cp **Chunk) Object {
	RT.vm = vm.context
	f, c := *fp, *cp
	f.lastOp = f.ip
	op := Opcode(c.Code[f.ip])
	f.ip++
	switch op {
	case OP_CONST:
		vm.Push(c.Constants[vm.readOperand(f, c)])
	case OP_NIL:
		vm.Push(NIL)
	case OP_TRUE:
		vm.Push(Boolean{B: true})
	case OP_FALSE:
		vm.Push(Boolean{B: false})
	case OP_POP:
		vm.Pop()
	case OP_DUP:
		vm.Push(vm.Peek(0))
	case OP_GET_LOCAL:
		vm.Push(capturedValue(vm.stack[f.slots+vm.readOperand(f, c)]))
	case OP_SET_LOCAL:
		slot := f.slots + vm.readOperand(f, c)
		if cell, ok := vm.stack[slot].(*bindingCell); ok {
			cell.Object = vm.Peek(0)
		} else {
			vm.stack[slot] = vm.Peek(0)
		}
	case OP_GET_UPVALUE:
		vm.Push(capturedValue(f.closure.upvalues[vm.readOperand(f, c)].closed))
	case OP_BIND_CELL:
		vm.stack[vm.stackTop-1] = &bindingCell{Object: vm.Peek(0)}
	case OP_GET_VAR:
		vm.Push(c.Constants[vm.readOperand(f, c)].(*Var).Resolve())
	case OP_SET_VAR:
		c.Constants[vm.readOperand(f, c)].(*Var).Value = vm.Peek(0)
	case OP_SET_VAR_META:
		vr := c.Constants[vm.readOperand(f, c)].(*Var)
		vr.meta = c.Constants[vm.readOperand(f, c)].(Map)
	case OP_MERGE_VAR_META:
		vr := c.Constants[vm.readOperand(f, c)].(*Var)
		vr.meta = vr.meta.Merge(vm.Pop().(Map))
	case OP_FINISH_DEF:
		vr := c.Constants[vm.readOperand(f, c)].(*Var)
		if vr.isMacro {
			vr.meta = vr.meta.Assoc(KEYWORDS.macro, Boolean{B: true}).(Map)
		}
		vm.Push(vr)
	case OP_WITH_META:
		obj := vm.Pop().(Meta)
		meta := vm.Pop().(Map)
		vm.Push(obj.WithMeta(meta))
	case OP_CHECK_CALLABLE:
		if _, ok := vm.Peek(0).(Callable); !ok {
			panic(RT.NewErrorWithPos(vm.Peek(0).ToString(false)+" is not a Fn", c.positionAt(f.lastOp)))
		}
	case OP_JUMP:
		offset := vm.readOperand(f, c)
		f.ip += offset
	case OP_JUMP_IF_FALSE:
		offset := vm.readOperand(f, c)
		if !ToBool(vm.Pop()) {
			f.ip += offset
		}
	case OP_LOOP:
		offset := vm.readOperand(f, c)
		f.ip -= offset
	case OP_CALL:
		argc := vm.readOperand(f, c)
		callee := vm.Peek(argc)
		if site := c.callSites[f.lastOp]; site != nil {
			previous := RT.currentExpr
			RT.currentExpr = site
			defer func() { RT.currentExpr = previous }()
		}
		if vm.callValue(callee, argc) {
			*fp = &vm.frames[vm.frameCount-1]
			*cp = (*fp).arityProto.Chunk
		}
	case OP_CLOSURE:
		proto := f.arityProto.SubFunctions[vm.readOperand(f, c)]
		fn := &Fn{proto: proto, isCompiled: true, upvalues: make([]*Upvalue, len(proto.Upvalues))}
		for i, u := range proto.Upvalues {
			var v Object
			if u.IsLocal {
				v = vm.stack[f.slots+u.Index]
			} else {
				v = f.closure.upvalues[u.Index].closed
			}
			if v == nil {
				panic(RT.NewError("VM invariant: uninitialized capture"))
			}
			fn.upvalues[i] = &Upvalue{closed: v}
		}
		vm.Push(fn)
	case OP_RETURN:
		result := vm.Pop()
		slots := f.slots
		vm.frameCount--
		vm.frames[vm.frameCount] = CallFrame{}
		vm.truncate(slots)
		if vm.frameCount == 0 {
			return result
		}
		vm.Push(result)
		*fp = &vm.frames[vm.frameCount-1]
		*cp = (*fp).arityProto.Chunk
	case OP_RECUR:
		argc := vm.readOperand(f, c)
		start := f.slots + vm.readOperand(f, c)
		copy(vm.stack[start:start+argc], vm.stack[vm.stackTop-argc:vm.stackTop])
		vm.truncate(start + argc)
	case OP_VECTOR:
		n := vm.readOperand(f, c)
		vm.Push(&ArrayVector{arr: vm.PopN(n)})
	case OP_MAP_NEW:
		n := vm.readOperand(f, c)
		if int64(n) > HASHMAP_THRESHOLD/2 {
			vm.Push(EmptyHashMap)
		} else {
			vm.Push(EmptyArrayMap())
		}
	case OP_MAP_CHECK:
		if m, ok := vm.Peek(1).(*HashMap); ok && m.containsKey(vm.Peek(0)) {
			panic(RT.NewError("Duplicate key: " + vm.Peek(0).ToString(false)))
		}
	case OP_MAP_ADD:
		v, k := vm.Pop(), vm.Pop()
		switch m := vm.Peek(0).(type) {
		case *HashMap:
			vm.stack[vm.stackTop-1] = m.Assoc(k, v)
		case *ArrayMap:
			if !m.Add(k, v) {
				panic(RT.NewError("Duplicate key: " + k.ToString(false)))
			}
		}
	case OP_SET_NEW:
		vm.Push(EmptySet())
	case OP_SET_ADD:
		v := vm.Pop()
		if !vm.Peek(0).(*MapSet).Add(v) {
			panic(RT.NewError("Duplicate set element: " + v.ToString(false)))
		}
	case OP_POPN:
		n := vm.readOperand(f, c)
		result := vm.Pop()
		vm.truncate(vm.stackTop - n)
		vm.Push(result)
	case OP_THROW:
		v := vm.Pop()
		if err, ok := v.(Error); ok {
			panic(err)
		}
		panic(RT.NewError("Cannot throw " + v.ToString(false)))
	case OP_TRY_BEGIN:
		idx := vm.readOperand(f, c)
		if vm.handlerCount == len(vm.handlers) {
			vm.handlers = append(vm.handlers, make([]ExceptionHandler, len(vm.handlers))...)
		}
		vm.handlers[vm.handlerCount] = ExceptionHandler{handlerIdx: idx, frameIndex: vm.frameCount - 1, stackTop: vm.stackTop}
		vm.handlerCount++
	case OP_TRY_END:
		vm.handlerCount--
		vm.handlers[vm.handlerCount] = ExceptionHandler{}
	case OP_FINALLY_END:
		if vm.pendingCount > 0 {
			p := vm.pending[vm.pendingCount-1]
			if p.frameIndex == vm.frameCount-1 && p.finishIP == f.lastOp {
				vm.pendingCount--
				vm.pending[vm.pendingCount] = pendingFinally{}
				panic(p.err)
			}
		}
	case OP_SET_MACRO:
		vr := c.Constants[vm.readOperand(f, c)].(*Var)
		vr.isMacro = true
		vr.isUsed = false
		if fn, ok := vr.Value.(*Fn); ok {
			fn.isMacro = true
		}
		setMacroMeta(vr)
		vm.Push(vr)
	default:
		panic(RT.NewError("Invalid opcode: " + strconv.Itoa(int(op))))
	}
	return nil
}

func (vm *VM) callValue(callee Object, argc int) bool {
	switch fn := callee.(type) {
	case *Fn:
		if !DISABLE_VM {
			fn.ensureCompiled()
		}
		if fn.isCompiled && fn.proto != nil {
			vm.callFn(fn, argc)
			return true
		}
		// This branch exists only for the explicitly selected AST test oracle.
		if !DISABLE_VM {
			panic(RT.NewError("VM invariant: uncompiled function"))
		}
		return vm.callOtherCallable(fn, argc)
	case Proc:
		if fn.Package == "" {
			base := vm.stackTop - argc - 1
			args := vm.stack[base+1 : vm.stackTop : vm.stackTop]
			vm.stackTop = base
			result := fn.Call(args)
			clear(vm.stack[base+1 : base+argc+1])
			vm.Push(result)
			return false
		}
		return vm.callOtherCallable(fn, argc)
	case Callable:
		return vm.callOtherCallable(fn, argc)
	default:
		panic(RT.NewError("Cannot call " + callee.ToString(false)))
	}
}
func (vm *VM) callOtherCallable(fn Callable, argc int) bool {
	args := vm.PopN(argc)
	vm.Pop()
	result := fn.Call(args)
	vm.Push(result)
	return false
}
func selectArityProto(proto *FunctionProto, argc int) *ArityProto {
	for _, a := range proto.Arities {
		if a.Arity == argc {
			return a
		}
	}
	if a := proto.VariadicArity; a != nil && argc >= a.Arity {
		return a
	}
	return nil
}
func buildArityErrorMessage(proto *FunctionProto, argc int) string {
	var expected []string
	for _, a := range proto.Arities {
		expected = append(expected, strconv.Itoa(a.Arity))
	}
	if a := proto.VariadicArity; a != nil {
		expected = append(expected, strconv.Itoa(a.Arity)+"+")
	}
	return "Wrong number of args (" + strconv.Itoa(argc) + ") passed to " + proto.Name + ", expected: " + strings.Join(expected, " or ")
}
func (vm *VM) callFn(fn *Fn, argc int) {
	proto := fn.proto
	a := selectArityProto(proto, argc)
	if len(proto.Arities) == 0 && proto.VariadicArity == nil && proto.Chunk != nil && argc == 0 {
		a = &ArityProto{Chunk: proto.Chunk, SubFunctions: proto.SubFunctions}
	}
	if a == nil {
		if fn.isMacro {
			fn.panicMacroArity(argc)
		}
		panic(RT.NewError(buildArityErrorMessage(proto, argc)))
	}
	if a.IsVariadic {
		n := argc - a.Arity
		if n > 0 {
			vm.Push(&ArraySeq{arr: vm.PopN(n)})
		} else {
			vm.Push(NIL)
		}
		argc = a.Arity + 1
	}
	if vm.frameCount == len(vm.frames) {
		vm.frames = append(vm.frames, make([]CallFrame, len(vm.frames))...)
	}
	vm.frames[vm.frameCount] = CallFrame{closure: fn, arityProto: a, slots: vm.stackTop - argc - 1}
	vm.frameCount++
}

// Runtime snapshots collect logical VM call frames only when needed (errors,
// asynchronous native work), avoiding per-call allocation and duplicate traces
// when an exception is rethrown through finally or another VM.
func (context *vmContext) appendTrace(stack *Callstack) {
	vm := context.vm
	if vm == nil {
		return
	}
	if context.parent != nil {
		context.parent.appendTrace(stack)
	}
	if site, ok := context.entry.(Traceable); ok {
		stack.pushFrame(Frame{traceable: site})
	}
	for i := 0; i+1 < vm.frameCount; i++ {
		f := &vm.frames[i]
		if site := f.arityProto.Chunk.callSites[f.lastOp]; site != nil {
			stack.pushFrame(Frame{traceable: site})
		}
	}
}
func (vm *VM) dispatchException(exc interface{}, fp **CallFrame, cp **Chunk) bool {
	for vm.handlerCount > 0 {
		vm.handlerCount--
		h := vm.handlers[vm.handlerCount]
		vm.handlers[vm.handlerCount] = ExceptionHandler{}
		for vm.pendingCount > 0 {
			p := vm.pending[vm.pendingCount-1]
			if h.frameIndex > p.frameIndex || (h.frameIndex == p.frameIndex && vm.handlerCount >= p.handlerDepth) {
				break
			}
			vm.pendingCount--
			vm.pending[vm.pendingCount] = pendingFinally{}
		}
		for vm.frameCount > h.frameIndex+1 {
			vm.frameCount--
			vm.frames[vm.frameCount] = CallFrame{}
		}
		f := &vm.frames[h.frameIndex]
		c := f.arityProto.Chunk
		info := &c.Handlers[h.handlerIdx]
		for _, catch := range info.Catches {
			if err, ok := exc.(Error); ok && IsInstance(catch.ExcType, err) {
				vm.truncate(h.stackTop)
				vm.Push(err)
				f.ip = catch.HandlerIP
				*fp = f
				*cp = c
				return true
			}
		}
		if info.FinallyIP >= 0 {
			if vm.pendingCount == len(vm.pending) {
				vm.pending = append(vm.pending, make([]pendingFinally, len(vm.pending))...)
			}
			vm.pending[vm.pendingCount] = pendingFinally{err: exc, frameIndex: h.frameIndex, finishIP: info.EndIP - 1, handlerDepth: vm.handlerCount}
			vm.pendingCount++
			vm.truncate(f.slots + info.TryLocalCount)
			vm.Push(NIL)
			f.ip = info.FinallyIP
			*fp = f
			*cp = c
			return true
		}
	}
	clear(vm.pending)
	vm.pendingCount = 0
	return false
}

var vmPool = sync.Pool{New: func() interface{} { return NewVM() }}

func VMExecute(fn *Fn, args []Object) Object {
	vm := vmPool.Get().(*VM)
	defer func() { vm.Reset(); vmPool.Put(vm) }()
	return vm.Execute(fn, args)
}
