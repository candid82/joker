package core

import (
	"strconv"
	"sync"
)

const (
	vmStackMax    = 256 * 64 // Maximum value stack size
	vmFramesMax   = 64       // Maximum call frame depth
	vmHandlersMax = 64       // Maximum exception handler depth
)

// Upvalue represents a captured variable in a closure.
type Upvalue struct {
	index  int      // Stack index while open (-1 when closed)
	closed Object   // Holds value after closing
	next   *Upvalue // Linked list for open upvalues
}

// CallFrame represents a single function invocation on the call stack.
type CallFrame struct {
	closure    *Fn         // The function being executed
	arityProto *ArityProto // The specific arity being executed
	ip         int         // Instruction pointer into chunk.Code
	slots      int         // Base index in VM stack for this frame's locals
}

// ExceptionHandler tracks an active try block for exception handling.
type ExceptionHandler struct {
	handlerIdx int // Index into chunk.Handlers
	frameIndex int // Frame index when installed
	stackTop   int // Stack top when try began
}

// VM is the bytecode virtual machine.
type VM struct {
	stack        []Object           // Value stack
	stackTop     int                // Points to next free slot
	frames       []CallFrame        // Call frame stack
	frameCount   int                // Number of active frames
	openUpvals   *Upvalue           // Linked list of open upvalues (sorted by stack index)
	handlers     []ExceptionHandler // Exception handler stack
	handlerCount int                // Number of active handlers
}

// NewVM creates a new VM instance.
func NewVM() *VM {
	return &VM{
		stack:    make([]Object, vmStackMax),
		frames:   make([]CallFrame, vmFramesMax),
		handlers: make([]ExceptionHandler, vmHandlersMax),
	}
}

// Reset clears the VM state for reuse.
func (vm *VM) Reset() {
	vm.stackTop = 0
	vm.frameCount = 0
	vm.openUpvals = nil
	vm.handlerCount = 0
}

// Push pushes a value onto the stack.
func (vm *VM) Push(value Object) {
	if vm.stackTop >= vmStackMax {
		panic(RT.NewError("VM stack overflow"))
	}
	if value == nil {
		panic(RT.NewError("VM BUG: pushing Go nil to stack"))
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
	result := vm.stack[vm.stackTop]
	if result == nil {
		panic(RT.NewError("VM BUG: popped Go nil from stack"))
	}
	return result
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
	proto := fn.proto
	argCount := len(args)

	// Select the appropriate arity
	var arityProto *ArityProto
	if len(proto.Arities) > 0 || proto.VariadicArity != nil {
		arityProto = selectArityProto(proto, argCount)
		if arityProto == nil {
			panic(RT.NewError(buildArityErrorMessage(proto, argCount)))
		}
	} else {
		// Legacy single-arity case
		if argCount != proto.Arity && !proto.Variadic {
			panic(RT.NewError("Expected " + strconv.Itoa(proto.Arity) + " arguments but got " + strconv.Itoa(argCount)))
		}
		arityProto = &ArityProto{
			Arity:        proto.Arity,
			IsVariadic:   proto.Variadic,
			Chunk:        proto.Chunk,
			Upvalues:     proto.Upvalues,
			SubFunctions: proto.SubFunctions,
		}
	}

	// Push the function onto the stack
	vm.Push(fn)

	// Push arguments
	if arityProto.IsVariadic {
		// Push fixed args
		for i := 0; i < arityProto.Arity && i < argCount; i++ {
			vm.Push(args[i])
		}
		// Pack rest args into ArraySeq
		restCount := argCount - arityProto.Arity
		if restCount > 0 {
			restArgs := make([]Object, restCount)
			for i := 0; i < restCount; i++ {
				restArgs[i] = args[arityProto.Arity+i]
			}
			vm.Push(&ArraySeq{arr: restArgs, index: 0})
		} else {
			vm.Push(NIL)
		}
		argCount = arityProto.Arity + 1
	} else {
		for _, arg := range args {
			vm.Push(arg)
		}
	}

	// Set up the initial call frame
	vm.frames[0] = CallFrame{
		closure:    fn,
		arityProto: arityProto,
		ip:         0,
		slots:      0,
	}
	vm.frameCount = 1

	return vm.run()
}

// ExecuteTopLevel executes a compiled zero-argument expression.
// This is used for top-level expressions in file evaluation.
// Unlike Execute(), this takes a FunctionProto directly (not an Fn).
func (vm *VM) ExecuteTopLevel(proto *FunctionProto) Object {
	// Reset state for this expression
	vm.stackTop = 0
	vm.frameCount = 0
	vm.openUpvals = nil
	vm.handlerCount = 0

	// Create a temporary Fn wrapper for the compiled expression
	fn := &Fn{proto: proto, isCompiled: true}

	// Push the function onto the stack (slot 0)
	vm.Push(fn)

	// Get the arity proto - Compile() uses legacy single-arity format
	var arityProto *ArityProto
	if len(proto.Arities) > 0 {
		arityProto = proto.Arities[0]
	} else {
		// Legacy format from Compile()
		arityProto = &ArityProto{
			Arity:        proto.Arity,
			IsVariadic:   proto.Variadic,
			Chunk:        proto.Chunk,
			Upvalues:     proto.Upvalues,
			SubFunctions: proto.SubFunctions,
		}
	}

	// Set up the call frame for zero-argument execution
	vm.frames[0] = CallFrame{
		closure:    fn,
		arityProto: arityProto,
		ip:         0,
		slots:      0,
	}
	vm.frameCount = 1

	return vm.run()
}

// vmReturnValue is used to signal a return from deep in the VM execution.
type vmReturnValue struct {
	result Object
}

// run is the main execution loop.
func (vm *VM) run() Object {
	frame := &vm.frames[vm.frameCount-1]
	chunk := frame.arityProto.Chunk

runLoop:
	for {
		// Set up panic recovery for exception handling
		didPanic := false
		var panicValue interface{}

		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
					panicValue = r
				}
			}()

			for {
				result := vm.executeOneOp(&frame, &chunk)
				if result != nil {
					panic(vmReturnValue{result: result})
				}
			}
		}()

		if didPanic {
			// Check if this is a return signal
			if ret, ok := panicValue.(vmReturnValue); ok {
				return ret.result
			}
			// Try to dispatch to an exception handler
			if err, ok := panicValue.(Error); ok {
				if vm.dispatchException(err, &frame, &chunk) {
					continue runLoop // Handler found, continue execution
				}
			}
			// No handler found or not an Error, re-panic
			panic(panicValue)
		}
	}
}

// executeOneOp executes a single opcode and returns non-nil if the VM should return.
func (vm *VM) executeOneOp(framePtr **CallFrame, chunkPtr **Chunk) Object {
	frame := *framePtr
	chunk := *chunkPtr

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
		var val Object
		if upval.index >= 0 {
			val = vm.stack[upval.index]
		} else {
			val = upval.closed
		}
		if val == nil {
			panic(RT.NewError("VM BUG: upvalue is nil"))
		}
		vm.Push(val)

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
		if v.Value == nil {
			panic(RT.NewError("Var not initialized: " + v.ToString(false)))
		}
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
		if vm.callValue(callee, argCount) {
			// New frame was pushed, switch to it
			*framePtr = &vm.frames[vm.frameCount-1]
			*chunkPtr = (*framePtr).arityProto.Chunk
		}
		// Otherwise, result is already on stack, continue in current frame

	case OP_CLOSURE:
		idx := vm.readShort(frame, chunk)
		proto := frame.arityProto.SubFunctions[idx]
		fn := &Fn{
			proto:      proto,
			upvalues:   make([]*Upvalue, len(proto.Upvalues)),
			isCompiled: true,
		}
		// Read upvalue info from bytecode and capture upvalues
		// We close all upvalues immediately to ensure the closure is self-contained
		// and can be called from any context (including a different VM execution).
		for i := range proto.Upvalues {
			isLocal := chunk.Code[frame.ip] == 1
			frame.ip++
			index := int(chunk.Code[frame.ip])
			frame.ip++

			var val Object
			if isLocal {
				// Capture from current frame's local
				val = vm.stack[frame.slots+index]
			} else {
				// Get from parent closure's upvalue
				parentUpval := frame.closure.upvalues[index]
				if parentUpval.index >= 0 {
					// Parent's upvalue is open, read from stack
					val = vm.stack[parentUpval.index]
				} else {
					// Parent's upvalue is closed, use its closed value
					val = parentUpval.closed
				}
			}
			if val == nil {
				panic(RT.NewError("VM BUG: captured nil value for upvalue"))
			}
			// Create a closed upvalue directly
			fn.upvalues[i] = &Upvalue{
				index:  -1, // Already closed
				closed: val,
			}
		}
		vm.Push(fn)

	case OP_RETURN:
		result := vm.Pop()
		vm.closeUpvalues(frame.slots)
		vm.frameCount--
		if vm.frameCount == 0 {
			return result // Signal to run() that we're done
		}
		vm.stackTop = frame.slots
		vm.Push(result)
		*framePtr = &vm.frames[vm.frameCount-1]
		*chunkPtr = (*framePtr).arityProto.Chunk

	case OP_RECUR:
		argCount := int(chunk.Code[frame.ip])
		frame.ip++
		slotStart := int(chunk.Code[frame.ip])
		frame.ip++
		// Copy new argument values to loop variable slots
		for i := 0; i < argCount; i++ {
			vm.stack[frame.slots+slotStart+i] = vm.stack[vm.stackTop-argCount+i]
		}
		vm.stackTop = frame.slots + slotStart + argCount
		// Don't reset frame.ip; let OP_LOOP handle the backward jump

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

	case OP_THROW:
		exc := vm.Pop()
		var err Error
		switch e := exc.(type) {
		case Error:
			err = e
		default:
			err = RT.NewError("Cannot throw " + exc.ToString(false))
		}
		// Just panic - the recover in run() will handle dispatch
		panic(err)

	case OP_TRY_BEGIN:
		handlerIdx := int(vm.readShort(frame, chunk))
		if vm.handlerCount >= vmHandlersMax {
			panic(RT.NewError("VM exception handler stack overflow"))
		}
		vm.handlers[vm.handlerCount] = ExceptionHandler{
			handlerIdx: handlerIdx,
			frameIndex: vm.frameCount - 1,
			stackTop:   vm.stackTop,
		}
		vm.handlerCount++

	case OP_TRY_END:
		// Normal exit from try block - pop the handler
		if vm.handlerCount > 0 {
			vm.handlerCount--
		}

	case OP_POP_SLOT:
		// Remove value at specific slot, shifting values above it down.
		// Used for catch scope cleanup to remove exception binding while
		// preserving operands for outer expressions.
		slot := int(chunk.Code[frame.ip])
		frame.ip++
		absoluteSlot := frame.slots + slot
		// Shift values above the slot down by 1
		for i := absoluteSlot; i < vm.stackTop-1; i++ {
			vm.stack[i] = vm.stack[i+1]
		}
		vm.stackTop--

	default:
		panic(RT.NewError("Unknown opcode: " + strconv.Itoa(int(op))))
	}

	return nil
}

// readShort reads a 2-byte big-endian value from the bytecode.
func (vm *VM) readShort(frame *CallFrame, chunk *Chunk) uint16 {
	hi := chunk.Code[frame.ip]
	lo := chunk.Code[frame.ip+1]
	frame.ip += 2
	return uint16(hi)<<8 | uint16(lo)
}

// callValue calls a callable with the given arguments.
// Returns true if a new call frame was pushed (for compiled functions),
// false if the call completed and the result is on the stack.
func (vm *VM) callValue(callee Object, argCount int) bool {
	switch fn := callee.(type) {
	case *Fn:
		if fn.isCompiled && fn.proto != nil {
			vm.callFn(fn, argCount)
			return true
		}
		// Fall back to AST evaluation for uncompiled/multi-arity functions
		args := make([]Object, argCount)
		for i := argCount - 1; i >= 0; i-- {
			args[i] = vm.Pop()
			if args[i] == nil {
				panic(RT.NewError("BUG: popped nil from VM stack"))
			}
		}
		vm.Pop() // Pop the function
		// Verify we have the correct function - sanity check
		if fn.isCompiled {
			panic(RT.NewError("BUG: isCompiled=true but proto=nil"))
		}
		result := fn.callAST(args) // Call AST directly to avoid re-checking isCompiled
		if result == nil {
			panic(RT.NewError("VM BUG: Fn.callAST returned Go nil"))
		}
		vm.Push(result)
		return false
	case Callable:
		// Native function or other callable
		args := make([]Object, argCount)
		for i := argCount - 1; i >= 0; i-- {
			args[i] = vm.Pop()
		}
		vm.Pop() // Pop the callable
		result := fn.Call(args)
		if result == nil {
			if obj, ok := fn.(Object); ok {
				panic(RT.NewError("VM BUG: Callable returned Go nil: " + obj.ToString(false)))
			}
			panic(RT.NewError("VM BUG: Callable returned Go nil"))
		}
		vm.Push(result)
		return false
	default:
		if callee == nil {
			panic(RT.NewError("Cannot call Go nil (var not initialized?)"))
		}
		panic(RT.NewError("Cannot call " + callee.GetType().ToString(false)))
	}
}

// selectArityProto selects the appropriate arity for the given argument count.
func selectArityProto(proto *FunctionProto, argCount int) *ArityProto {
	// Try exact match in fixed arities
	for _, arity := range proto.Arities {
		if arity.Arity == argCount {
			return arity
		}
	}
	// Try variadic (accepts Arity or more args)
	if proto.VariadicArity != nil && argCount >= proto.VariadicArity.Arity {
		return proto.VariadicArity
	}
	return nil
}

// buildArityErrorMessage builds an error message for arity mismatch.
func buildArityErrorMessage(proto *FunctionProto, argCount int) string {
	name := proto.Name
	if name == "" {
		name = "function"
	}

	// Collect valid arities
	var arities []int
	for _, a := range proto.Arities {
		arities = append(arities, a.Arity)
	}
	if proto.VariadicArity != nil {
		arities = append(arities, proto.VariadicArity.Arity)
	}

	// Legacy single-arity case
	if len(arities) == 0 {
		return "Wrong number of args (" + strconv.Itoa(argCount) + ") passed to " + name
	}

	// Build expected arities string
	expected := ""
	for i, a := range arities {
		if i > 0 {
			if i == len(arities)-1 {
				expected += " or "
			} else {
				expected += ", "
			}
		}
		expected += strconv.Itoa(a)
		if proto.VariadicArity != nil && a == proto.VariadicArity.Arity {
			expected += "+"
		}
	}

	return "Wrong number of args (" + strconv.Itoa(argCount) + ") passed to " + name + ", expected: " + expected
}

// callFn sets up a call frame for a compiled function.
func (vm *VM) callFn(fn *Fn, argCount int) {
	proto := fn.proto

	// Select the appropriate arity
	var arityProto *ArityProto

	// Check if we have new multi-arity structure
	if len(proto.Arities) > 0 || proto.VariadicArity != nil {
		arityProto = selectArityProto(proto, argCount)
		if arityProto == nil {
			panic(RT.NewError(buildArityErrorMessage(proto, argCount)))
		}
	} else {
		// Legacy single-arity case
		if argCount != proto.Arity && !proto.Variadic {
			panic(RT.NewError("Expected " + strconv.Itoa(proto.Arity) + " arguments but got " + strconv.Itoa(argCount)))
		}
		// Create a temporary ArityProto for legacy case
		arityProto = &ArityProto{
			Arity:        proto.Arity,
			IsVariadic:   proto.Variadic,
			Chunk:        proto.Chunk,
			Upvalues:     proto.Upvalues,
			SubFunctions: proto.SubFunctions,
		}
	}

	// Pack rest args for variadic calls
	if arityProto.IsVariadic {
		restCount := argCount - arityProto.Arity
		if restCount > 0 {
			// Collect rest args from stack into ArraySeq
			restArgs := make([]Object, restCount)
			for i := restCount - 1; i >= 0; i-- {
				restArgs[i] = vm.Pop()
			}
			vm.Push(&ArraySeq{arr: restArgs, index: 0})
			argCount = arityProto.Arity + 1 // fixed args + rest seq
		} else {
			// No rest args, push NIL for the rest param
			vm.Push(NIL)
			argCount = arityProto.Arity + 1
		}
	}

	if vm.frameCount >= vmFramesMax {
		panic(RT.NewError("VM call stack overflow"))
	}

	frame := &vm.frames[vm.frameCount]
	vm.frameCount++
	frame.closure = fn
	frame.arityProto = arityProto
	frame.ip = 0
	frame.slots = vm.stackTop - argCount - 1
}

// closeUpvalues closes all upvalues at or above the given stack index.
func (vm *VM) closeUpvalues(lastIndex int) {
	for vm.openUpvals != nil && vm.openUpvals.index >= lastIndex {
		upval := vm.openUpvals
		val := vm.stack[upval.index]
		if val == nil {
			panic(RT.NewError("VM BUG: closing upvalue with nil value at index " + strconv.Itoa(upval.index)))
		}
		upval.closed = val
		upval.index = -1 // Mark as closed
		vm.openUpvals = upval.next
	}
}

// dispatchException tries to find a matching catch handler for the exception.
// Returns true if a handler was found and execution should continue,
// false if no handler was found and the exception should propagate.
func (vm *VM) dispatchException(exc Error, framePtr **CallFrame, chunkPtr **Chunk) bool {
	for vm.handlerCount > 0 {
		vm.handlerCount--
		handler := vm.handlers[vm.handlerCount]

		// Get the frame where the handler was installed
		if handler.frameIndex >= vm.frameCount {
			continue // Frame was already popped, skip this handler
		}

		// Restore to the handler's frame
		for vm.frameCount > handler.frameIndex+1 {
			vm.closeUpvalues(vm.frames[vm.frameCount-1].slots)
			vm.frameCount--
		}

		frame := &vm.frames[handler.frameIndex]
		chunk := frame.arityProto.Chunk
		handlerInfo := &chunk.Handlers[handler.handlerIdx]

		// Find a matching catch clause
		for _, catchInfo := range handlerInfo.Catches {
			if IsInstance(catchInfo.ExcType, exc) {
				// Found a match! Place exception at the correct slot.
				// The exception binding slot is TryLocalCount (where localCount was at try-begin).
				// Any operands on the stack (between locals and stackTop) need to be shifted up
				// to make room for the exception at its expected slot.
				excSlot := frame.slots + handlerInfo.TryLocalCount

				// Shift operands up by 1 to make room for exception
				for i := handler.stackTop - 1; i >= excSlot; i-- {
					vm.stack[i+1] = vm.stack[i]
				}

				// Place exception at the correct slot
				vm.stack[excSlot] = exc
				vm.stackTop = handler.stackTop + 1

				frame.ip = catchInfo.HandlerIP

				// Update the caller's frame/chunk pointers
				*framePtr = frame
				*chunkPtr = chunk
				return true
			}
		}

		// No matching catch - check for finally
		if handlerInfo.FinallyIP >= 0 {
			// Execute finally by jumping to it
			// The exception will be re-thrown after finally completes
			// For now, we'll re-panic (finally execution is complex)
			// TODO: Implement proper finally execution with pending exception
		}
	}

	// No handler found
	return false
}

// vmPool is a pool of VM instances for reuse, avoiding allocations
// on each call to higher-order functions like map/reduce.
var vmPool = sync.Pool{
	New: func() interface{} {
		return NewVM()
	},
}

// VMExecute executes a compiled function.
// Uses a pool of VM instances to avoid allocating a new VM on each call.
func VMExecute(fn *Fn, args []Object) Object {
	vm := vmPool.Get().(*VM)
	result := vm.Execute(fn, args)
	vmPool.Put(vm)
	return result
}
