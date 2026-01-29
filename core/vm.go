package core

import (
	"strconv"
)

const (
	vmStackMax  = 256 * 64 // Maximum value stack size
	vmFramesMax = 64       // Maximum call frame depth
)

// Upvalue represents a captured variable in a closure.
type Upvalue struct {
	index  int       // Stack index while open (-1 when closed)
	closed Object    // Holds value after closing
	next   *Upvalue  // Linked list for open upvalues
}

// CallFrame represents a single function invocation on the call stack.
type CallFrame struct {
	closure *Fn   // The function being executed
	ip      int   // Instruction pointer into chunk.Code
	slots   int   // Base index in VM stack for this frame's locals
}

// VM is the bytecode virtual machine.
type VM struct {
	stack      []Object     // Value stack
	stackTop   int          // Points to next free slot
	frames     []CallFrame  // Call frame stack
	frameCount int          // Number of active frames
	openUpvals *Upvalue     // Linked list of open upvalues (sorted by stack index)
}

// NewVM creates a new VM instance.
func NewVM() *VM {
	return &VM{
		stack:  make([]Object, vmStackMax),
		frames: make([]CallFrame, vmFramesMax),
	}
}

// Reset clears the VM state for reuse.
func (vm *VM) Reset() {
	vm.stackTop = 0
	vm.frameCount = 0
	vm.openUpvals = nil
}

// Push pushes a value onto the stack.
func (vm *VM) Push(value Object) {
	if vm.stackTop >= vmStackMax {
		panic(RT.NewError("VM stack overflow"))
	}
	vm.stack[vm.stackTop] = value
	vm.stackTop++
}

// Pop pops a value from the stack.
func (vm *VM) Pop() Object {
	if vm.stackTop == 0 {
		panic(RT.NewError("VM stack underflow"))
	}
	vm.stackTop--
	return vm.stack[vm.stackTop]
}

// Peek returns a value from the stack without popping.
// distance 0 = top of stack.
func (vm *VM) Peek(distance int) Object {
	return vm.stack[vm.stackTop-1-distance]
}

// PopN pops n values from the stack and returns them in order (first popped = last in slice).
func (vm *VM) PopN(n int) []Object {
	args := make([]Object, n)
	for i := n - 1; i >= 0; i-- {
		args[i] = vm.Pop()
	}
	return args
}

// Execute runs a compiled function with the given arguments.
func (vm *VM) Execute(fn *Fn, args []Object) Object {
	vm.Reset()

	// Push the function and arguments onto the stack
	vm.Push(fn)
	for _, arg := range args {
		vm.Push(arg)
	}

	// Set up the initial call frame
	vm.frames[0] = CallFrame{
		closure: fn,
		ip:      0,
		slots:   0,
	}
	vm.frameCount = 1

	return vm.run()
}

// run is the main execution loop.
func (vm *VM) run() Object {
	frame := &vm.frames[vm.frameCount-1]
	chunk := frame.closure.proto.Chunk

	for {
		op := Opcode(chunk.Code[frame.ip])
		frame.ip++

		switch op {
		case OP_CONST:
			idx := vm.readShort(frame, chunk)
			vm.Push(chunk.Constants[idx])

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
			slot := chunk.Code[frame.ip]
			frame.ip++
			vm.Push(vm.stack[frame.slots+int(slot)])

		case OP_SET_LOCAL:
			slot := chunk.Code[frame.ip]
			frame.ip++
			vm.stack[frame.slots+int(slot)] = vm.Peek(0)

		case OP_GET_UPVALUE:
			slot := chunk.Code[frame.ip]
			frame.ip++
			upval := frame.closure.upvalues[slot]
			if upval.index >= 0 {
				vm.Push(vm.stack[upval.index])
			} else {
				vm.Push(upval.closed)
			}

		case OP_SET_UPVALUE:
			slot := chunk.Code[frame.ip]
			frame.ip++
			upval := frame.closure.upvalues[slot]
			if upval.index >= 0 {
				vm.stack[upval.index] = vm.Peek(0)
			} else {
				upval.closed = vm.Peek(0)
			}

		case OP_GET_VAR:
			idx := vm.readShort(frame, chunk)
			v := chunk.Constants[idx].(*Var)
			vm.Push(v.Value)

		case OP_SET_VAR:
			idx := vm.readShort(frame, chunk)
			v := chunk.Constants[idx].(*Var)
			v.Value = vm.Peek(0)

		case OP_ADD:
			b := EnsureObjectIsNumber(vm.Pop(), "")
			a := EnsureObjectIsNumber(vm.Pop(), "")
			ops := GetOps(a).Combine(GetOps(b))
			vm.Push(ops.Add(a, b))

		case OP_SUBTRACT:
			b := EnsureObjectIsNumber(vm.Pop(), "")
			a := EnsureObjectIsNumber(vm.Pop(), "")
			ops := GetOps(a).Combine(GetOps(b))
			vm.Push(ops.Subtract(a, b))

		case OP_MULTIPLY:
			b := EnsureObjectIsNumber(vm.Pop(), "")
			a := EnsureObjectIsNumber(vm.Pop(), "")
			ops := GetOps(a).Combine(GetOps(b))
			vm.Push(ops.Multiply(a, b))

		case OP_DIVIDE:
			b := EnsureObjectIsNumber(vm.Pop(), "")
			a := EnsureObjectIsNumber(vm.Pop(), "")
			ops := GetOps(a).Combine(GetOps(b))
			vm.Push(ops.Divide(a, b))

		case OP_NEGATE:
			val := EnsureObjectIsNumber(vm.Pop(), "")
			switch v := val.(type) {
			case Int:
				vm.Push(Int{I: -v.I})
			case Double:
				vm.Push(Double{D: -v.D})
			default:
				// For BigInt/BigFloat/Ratio, use multiply by -1
				ops := GetOps(val)
				vm.Push(ops.Multiply(val, Int{I: -1}))
			}

		case OP_EQUAL:
			b := vm.Pop()
			a := vm.Pop()
			vm.Push(Boolean{B: a.Equals(b)})

		case OP_LESS:
			b := EnsureObjectIsNumber(vm.Pop(), "")
			a := EnsureObjectIsNumber(vm.Pop(), "")
			ops := GetOps(a).Combine(GetOps(b))
			vm.Push(Boolean{B: ops.Lt(a, b)})

		case OP_GREATER:
			b := EnsureObjectIsNumber(vm.Pop(), "")
			a := EnsureObjectIsNumber(vm.Pop(), "")
			ops := GetOps(a).Combine(GetOps(b))
			vm.Push(Boolean{B: ops.Gt(a, b)})

		case OP_NOT:
			val := vm.Pop()
			vm.Push(Boolean{B: !ToBool(val)})

		case OP_JUMP:
			offset := vm.readShort(frame, chunk)
			frame.ip += int(int16(offset))

		case OP_JUMP_IF_FALSE:
			offset := vm.readShort(frame, chunk)
			if !ToBool(vm.Peek(0)) {
				frame.ip += int(int16(offset))
			}
			vm.Pop()

		case OP_LOOP:
			offset := vm.readShort(frame, chunk)
			frame.ip -= int(offset)

		case OP_CALL:
			argCount := int(chunk.Code[frame.ip])
			frame.ip++
			callee := vm.Peek(argCount)
			vm.callValue(callee, argCount)
			frame = &vm.frames[vm.frameCount-1]
			chunk = frame.closure.proto.Chunk

		case OP_CLOSURE:
			idx := vm.readShort(frame, chunk)
			parentProto := frame.closure.proto
			proto := parentProto.SubFunctions[idx]
			fn := &Fn{
				proto:      proto,
				upvalues:   make([]*Upvalue, len(proto.Upvalues)),
				isCompiled: true,
			}
			for i, info := range proto.Upvalues {
				if info.IsLocal {
					fn.upvalues[i] = vm.captureUpvalue(frame.slots + int(info.Index))
				} else {
					fn.upvalues[i] = frame.closure.upvalues[info.Index]
				}
			}
			vm.Push(fn)

		case OP_RETURN:
			result := vm.Pop()
			vm.closeUpvalues(frame.slots)
			vm.frameCount--
			if vm.frameCount == 0 {
				return result
			}
			vm.stackTop = frame.slots
			vm.Push(result)
			frame = &vm.frames[vm.frameCount-1]
			chunk = frame.closure.proto.Chunk

		case OP_RECUR:
			argCount := int(chunk.Code[frame.ip])
			frame.ip++
			// Copy new argument values to local slots
			for i := 0; i < argCount; i++ {
				vm.stack[frame.slots+1+i] = vm.stack[vm.stackTop-argCount+i]
			}
			vm.stackTop = frame.slots + 1 + argCount
			frame.ip = 0 // Jump back to start of function

		case OP_VECTOR:
			count := int(vm.readShort(frame, chunk))
			elements := make([]Object, count)
			for i := count - 1; i >= 0; i-- {
				elements[i] = vm.Pop()
			}
			vm.Push(NewVectorFrom(elements...))

		case OP_MAP:
			count := int(vm.readShort(frame, chunk))
			m := EmptyArrayMap()
			for i := 0; i < count; i++ {
				val := vm.Pop()
				key := vm.Pop()
				m = m.Assoc(key, val).(*ArrayMap)
			}
			vm.Push(m)

		case OP_SET:
			count := int(vm.readShort(frame, chunk))
			elements := make([]Object, count)
			for i := count - 1; i >= 0; i-- {
				elements[i] = vm.Pop()
			}
			vm.Push(NewSetFromSeq(NewListFrom(elements...)))

		case OP_CLOSE_UPVALUE:
			vm.closeUpvalues(vm.stackTop - 1)
			vm.Pop()

		case OP_POPN:
			// Pop N values from under the top (preserves top)
			n := int(chunk.Code[frame.ip])
			frame.ip++
			if n > 0 {
				top := vm.stack[vm.stackTop-1]
				vm.stackTop -= n + 1 // Remove n values plus old top position
				vm.stack[vm.stackTop] = top
				vm.stackTop++
			}

		default:
			panic(RT.NewError("Unknown opcode: " + strconv.Itoa(int(op))))
		}
	}
}

// readShort reads a 2-byte big-endian value from the bytecode.
func (vm *VM) readShort(frame *CallFrame, chunk *Chunk) uint16 {
	hi := chunk.Code[frame.ip]
	lo := chunk.Code[frame.ip+1]
	frame.ip += 2
	return uint16(hi)<<8 | uint16(lo)
}

// callValue calls a callable with the given arguments.
func (vm *VM) callValue(callee Object, argCount int) {
	switch fn := callee.(type) {
	case *Fn:
		if fn.isCompiled && fn.proto != nil {
			vm.callFn(fn, argCount)
		} else {
			// Fall back to AST evaluation
			args := make([]Object, argCount)
			for i := argCount - 1; i >= 0; i-- {
				args[i] = vm.Pop()
			}
			vm.Pop() // Pop the function
			result := fn.Call(args)
			vm.Push(result)
		}
	case Callable:
		// Native function or other callable
		args := make([]Object, argCount)
		for i := argCount - 1; i >= 0; i-- {
			args[i] = vm.Pop()
		}
		vm.Pop() // Pop the callable
		result := fn.Call(args)
		vm.Push(result)
	default:
		panic(RT.NewError("Cannot call " + callee.GetType().ToString(false)))
	}
}

// callFn sets up a call frame for a compiled function.
func (vm *VM) callFn(fn *Fn, argCount int) {
	proto := fn.proto
	if argCount != proto.Arity && !proto.Variadic {
		panic(RT.NewError("Expected " + strconv.Itoa(proto.Arity) + " arguments but got " + strconv.Itoa(argCount)))
	}

	if vm.frameCount >= vmFramesMax {
		panic(RT.NewError("VM call stack overflow"))
	}

	frame := &vm.frames[vm.frameCount]
	vm.frameCount++
	frame.closure = fn
	frame.ip = 0
	frame.slots = vm.stackTop - argCount - 1
}

// captureUpvalue captures a local variable for a closure.
func (vm *VM) captureUpvalue(stackIndex int) *Upvalue {
	var prev *Upvalue
	upval := vm.openUpvals

	// Find insertion point (sorted by stack index, descending)
	for upval != nil && upval.index > stackIndex {
		prev = upval
		upval = upval.next
	}

	// Reuse existing upvalue if found
	if upval != nil && upval.index == stackIndex {
		return upval
	}

	// Create new upvalue
	newUpval := &Upvalue{
		index: stackIndex,
		next:  upval,
	}

	if prev == nil {
		vm.openUpvals = newUpval
	} else {
		prev.next = newUpval
	}

	return newUpval
}

// closeUpvalues closes all upvalues at or above the given stack index.
func (vm *VM) closeUpvalues(lastIndex int) {
	for vm.openUpvals != nil && vm.openUpvals.index >= lastIndex {
		upval := vm.openUpvals
		upval.closed = vm.stack[upval.index]
		upval.index = -1 // Mark as closed
		vm.openUpvals = upval.next
	}
}

// Global VM instance for simple cases
var globalVM = NewVM()

// VMExecute executes a compiled function using the global VM.
func VMExecute(fn *Fn, args []Object) Object {
	return globalVM.Execute(fn, args)
}
