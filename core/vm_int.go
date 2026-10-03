package core

import "reflect"

// The object stack remains GC-visible. Only fresh integer results use this
// private marker and the parallel, pointer-free payload storage. Markers never
// cross an Object interface exposed to Joker/native code.
type vmIntegerMarker struct{ Object }

var vmIntegerObject Object = &vmIntegerMarker{}
var integerIncPC, integerEqualsPC uintptr // initialized under the execution GIL

func isCallOpcode(op Opcode) bool {
	return op == OP_CALL || op == OP_CALL_INT_INC || op == OP_CALL_INT_EQ
}

func (vm *VM) pushInteger(value int) {
	vm.ensureStack(vm.stackTop + 1)
	vm.stack[vm.stackTop] = vmIntegerObject
	vm.integers[vm.stackTop] = value
	vm.stackTop++
}

func (vm *VM) pushSlot(slot int) {
	if _, ok := vm.stack[slot].(*vmIntegerMarker); ok {
		vm.pushInteger(vm.integers[slot])
	} else {
		vm.Push(capturedValue(vm.stack[slot]))
	}
}

func (vm *VM) slotObject(slot int) Object {
	if _, ok := vm.stack[slot].(*vmIntegerMarker); ok {
		vm.stack[slot] = boxInt(vm.integers[slot])
		vm.integers[slot] = 0
	}
	return vm.stack[slot]
}

func (vm *VM) slotInteger(slot int) (int, bool) {
	switch value := vm.stack[slot].(type) {
	case *vmIntegerMarker:
		return vm.integers[slot], true
	case Int:
		return value.I, true
	default:
		return 0, false
	}
}

func (vm *VM) popValue() (Object, int) {
	if vm.stackTop == 0 {
		panic(RT.NewError("VM stack underflow"))
	}
	vm.stackTop--
	value, integer := vm.stack[vm.stackTop], vm.integers[vm.stackTop]
	vm.stack[vm.stackTop], vm.integers[vm.stackTop] = nil, 0
	if value == nil {
		panic(RT.NewError("VM invariant: nil stack slot"))
	}
	return value, integer
}

func (vm *VM) pushValue(value Object, integer int) {
	if _, ok := value.(*vmIntegerMarker); ok {
		vm.pushInteger(integer)
	} else {
		vm.Push(value)
	}
}

func integerObject(value Object, integer int) Object {
	if _, ok := value.(*vmIntegerMarker); ok {
		return boxInt(integer)
	}
	return value
}

// Recognize an exact leaf wrapper, rather than trusting a function's name or
// declared return type. The helper's current implementation is checked on every
// call, after argument evaluation. Bytecode is immutable during execution.
func (a *ArityProto) integerHelper() *Var {
	if a.integerWrapperChecked {
		return a.integerWrapperHelper
	}
	a.integerWrapperChecked = true
	if a.IsVariadic || a.Arity < 1 || a.Arity > 2 || a.Chunk == nil {
		return nil
	}
	code := a.Chunk.Code
	if len(code) != 12+5*a.Arity || Opcode(code[0]) != OP_GET_VAR || Opcode(code[5]) != OP_CHECK_CALLABLE {
		return nil
	}
	index := operandAt(code, 1)
	if index < 0 || index >= len(a.Chunk.Constants) {
		return nil
	}
	helper, ok := a.Chunk.Constants[index].(*Var)
	if !ok {
		return nil
	}
	ip := 6
	for i := 1; i <= a.Arity; i++ {
		if Opcode(code[ip]) != OP_GET_LOCAL || operandAt(code, ip+1) != i {
			return nil
		}
		ip += 5
	}
	if Opcode(code[ip]) != OP_CALL || operandAt(code, ip+1) != a.Arity || Opcode(code[ip+5]) != OP_RETURN {
		return nil
	}
	a.integerWrapperHelper = helper
	return helper
}

func integerImplementation(callee Object, argc int, op Opcode) bool {
	fn, ok := callee.(*Fn)
	if !ok || !fn.isCompiled || fn.proto == nil {
		return false
	}
	arity := selectArityProto(fn.proto, argc)
	if arity == nil {
		return false
	}
	helper := arity.integerHelper()
	if helper == nil {
		return false
	}
	proc, ok := helper.Resolve().(Proc)
	if !ok || proc.Package != "" || proc.InExecution != nil || proc.Fn == nil {
		return false
	}
	if integerIncPC == 0 {
		integerIncPC = reflect.ValueOf(procInc).Pointer()
		integerEqualsPC = reflect.ValueOf(procEquals).Pointer()
	}
	pc := integerIncPC
	if op == OP_CALL_INT_EQ {
		pc = integerEqualsPC
	}
	return reflect.ValueOf(proc.Fn).Pointer() == pc
}

func (vm *VM) integerCall(callee Object, argc int, op Opcode) bool {
	base := vm.stackTop - argc - 1
	x, ok := vm.slotInteger(base + 1)
	if !ok {
		return false
	}
	var y int
	if argc == 2 {
		y, ok = vm.slotInteger(base + 2)
		if !ok {
			return false
		}
	}
	if !integerImplementation(callee, argc, op) {
		return false
	}
	vm.truncate(base)
	if op == OP_CALL_INT_INC {
		vm.pushInteger(x + 1)
	} else {
		vm.Push(boxBoolean(x == y))
	}
	return true
}
