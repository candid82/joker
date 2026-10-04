package core

import "fmt"

// ValidateFunctionProto checks instruction boundaries, operands, control-flow
// stack shapes, captures, and exception entries before packed code can execute.
// Packed data is internal trusted build output, not a sandboxed public format.
func ValidateFunctionProto(proto *FunctionProto) error {
	seen := make(map[*FunctionProto]bool)
	var visit func(*FunctionProto) error
	visit = func(p *FunctionProto) error {
		if p == nil {
			return fmt.Errorf("nil function prototype")
		}
		if seen[p] {
			return fmt.Errorf("cyclic function prototype")
		}
		seen[p] = true
		defer delete(seen, p)
		arities := append([]*ArityProto(nil), p.Arities...)
		if p.VariadicArity != nil {
			arities = append(arities, p.VariadicArity)
		}
		if p.Chunk != nil {
			if len(arities) != 0 {
				return fmt.Errorf("function has both top-level body and arities")
			}
			arities = append(arities, &ArityProto{Chunk: p.Chunk, SubFunctions: p.SubFunctions})
		}
		if len(arities) == 0 {
			return fmt.Errorf("function has no body")
		}
		for _, u := range p.Upvalues {
			if u.Index < 0 {
				return fmt.Errorf("negative capture index")
			}
		}
		for _, a := range arities {
			if a == nil || a.Arity < 0 {
				return fmt.Errorf("invalid arity")
			}
			if err := validateChunk(p, a); err != nil {
				return fmt.Errorf("%s: %w", p.Name, err)
			}
			for _, sub := range a.SubFunctions {
				if err := visit(sub); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(proto)
}

func validateChunk(p *FunctionProto, a *ArityProto) error {
	c := a.Chunk
	if c == nil || len(c.Code) == 0 {
		return fmt.Errorf("empty bytecode")
	}
	if len(c.Positions) != len(c.Code) {
		return fmt.Errorf("invalid source position table")
	}
	boundaries := make(map[int]bool)
	for ip := 0; ip < len(c.Code); {
		op := Opcode(c.Code[ip])
		n, ok := opcodeOperands(op)
		if !ok || ip+1+4*n > len(c.Code) {
			return fmt.Errorf("invalid instruction at %d", ip)
		}
		boundaries[ip] = true
		ip += 1 + 4*n
	}
	for ip, site := range c.callSites {
		if site != nil && (!boundaries[ip] || Opcode(c.Code[ip]) != OP_CALL) {
			return fmt.Errorf("invalid call site %d", ip)
		}
	}
	for _, h := range c.Handlers {
		if h.TryLocalCount < 1 || h.EndIP < 0 || h.EndIP > len(c.Code) || (h.EndIP < len(c.Code) && !boundaries[h.EndIP]) {
			return fmt.Errorf("invalid handler end")
		}
		if h.FinallyIP != -1 && (!boundaries[h.FinallyIP] || h.EndIP <= h.FinallyIP || Opcode(c.Code[h.EndIP-1]) != OP_FINALLY_END) {
			return fmt.Errorf("invalid finally target")
		}
		for _, catch := range h.Catches {
			if !boundaries[catch.HandlerIP] || catch.ExcType == nil || catch.LocalSlot != h.TryLocalCount {
				return fmt.Errorf("invalid catch target")
			}
		}
	}
	// Validate operand references even in unreachable code.
	for ip := range boundaries {
		op := Opcode(c.Code[ip])
		n, _ := opcodeOperands(op)
		v := 0
		if n > 0 {
			v = operandAt(c.Code, ip+1)
		}
		switch op {
		case OP_CONST, OP_GET_VAR, OP_SET_VAR, OP_SET_VAR_META, OP_MERGE_VAR_META, OP_FINISH_DEF, OP_SET_MACRO:
			if v < 0 || v >= len(c.Constants) || c.Constants[v] == nil {
				return fmt.Errorf("invalid constant at %d", ip)
			}
			if op != OP_CONST {
				if _, ok := c.Constants[v].(*Var); !ok {
					return fmt.Errorf("non-Var operand at %d", ip)
				}
			}
			if op == OP_SET_VAR_META {
				idx := operandAt(c.Code, ip+5)
				if idx < 0 || idx >= len(c.Constants) {
					return fmt.Errorf("invalid metadata constant")
				}
				if _, ok := c.Constants[idx].(Map); !ok {
					return fmt.Errorf("non-map metadata constant")
				}
			}
		case OP_GET_UPVALUE:
			if v < 0 || v >= len(p.Upvalues) {
				return fmt.Errorf("invalid upvalue at %d", ip)
			}
		case OP_CLOSURE:
			if v < 0 || v >= len(a.SubFunctions) || a.SubFunctions[v] == nil {
				return fmt.Errorf("invalid subfunction at %d", ip)
			}
		case OP_TRY_BEGIN:
			if v < 0 || v >= len(c.Handlers) {
				return fmt.Errorf("invalid handler at %d", ip)
			}
		case OP_JUMP, OP_JUMP_IF_FALSE, OP_LOOP:
			target := ip + 5 + v
			if op == OP_LOOP {
				target = ip + 5 - v
			}
			if !boundaries[target] {
				return fmt.Errorf("invalid jump target at %d", ip)
			}
		}
	}
	type stackState struct{ values, handlers int }
	depths := make(map[int]stackState)
	work := []int{}
	enqueue := func(ip int, state stackState) error {
		if !boundaries[ip] {
			return fmt.Errorf("execution falls outside bytecode at %d", ip)
		}
		if old, ok := depths[ip]; ok {
			if old != state {
				return fmt.Errorf("inconsistent stack state at %d: %v vs %v", ip, old, state)
			}
			return nil
		}
		depths[ip] = state
		work = append(work, ip)
		return nil
	}
	initial := a.Arity + 1
	if a.IsVariadic {
		initial++
	}
	if err := enqueue(0, stackState{values: initial}); err != nil {
		return err
	}
	for len(work) > 0 {
		ip := work[len(work)-1]
		work = work[:len(work)-1]
		depth, handlers := depths[ip].values, depths[ip].handlers
		op := Opcode(c.Code[ip])
		n, _ := opcodeOperands(op)
		next := ip + 1 + 4*n
		v := 0
		if n > 0 {
			v = operandAt(c.Code, ip+1)
		}
		need, delta := 0, 0
		switch op {
		case OP_CONST, OP_NIL, OP_TRUE, OP_FALSE, OP_GET_UPVALUE, OP_GET_VAR, OP_FINISH_DEF, OP_SET_MACRO, OP_MAP_NEW, OP_SET_NEW:
			delta = 1
		case OP_GET_LOCAL:
			if v < 0 || v >= depth {
				return fmt.Errorf("invalid local at %d", ip)
			}
			delta = 1
		case OP_SET_LOCAL:
			if v < 0 || v >= depth-1 {
				return fmt.Errorf("invalid local store at %d", ip)
			}
			need = 1
		case OP_POP:
			need, delta = 1, -1
		case OP_DUP:
			need, delta = 1, 1
		case OP_BIND_CELL, OP_CHECK_CALLABLE, OP_SET_VAR:
			need = 1
		case OP_WITH_META:
			need, delta = 2, -1
		case OP_MERGE_VAR_META:
			need, delta = 1, -1
		case OP_CALL:
			need, delta = v+1, -v
		case OP_VECTOR:
			need, delta = v, 1-v
		case OP_MAP_CHECK:
			need = 2
		case OP_MAP_ADD:
			need, delta = 3, -2
		case OP_SET_ADD:
			need, delta = 2, -1
		case OP_POPN:
			need, delta = v+1, -v
		case OP_CLOSURE:
			for _, u := range a.SubFunctions[v].Upvalues {
				if u.Index < 0 || (u.IsLocal && u.Index >= depth) || (!u.IsLocal && u.Index >= len(p.Upvalues)) {
					return fmt.Errorf("invalid closure capture at %d", ip)
				}
			}
			delta = 1
		case OP_RECUR:
			slot := operandAt(c.Code, ip+5)
			if slot < 1 || slot+v > depth-v {
				return fmt.Errorf("invalid recur operands at %d", ip)
			}
			need = v
			delta = slot + v - depth
		case OP_TRY_BEGIN:
			h := c.Handlers[v]
			if h.TryLocalCount > depth {
				return fmt.Errorf("invalid handler stack depth")
			}
			for _, catch := range h.Catches {
				if err := enqueue(catch.HandlerIP, stackState{h.TryLocalCount + 1, handlers}); err != nil {
					return err
				}
			}
			if h.FinallyIP >= 0 {
				if err := enqueue(h.FinallyIP, stackState{h.TryLocalCount + 1, handlers}); err != nil {
					return err
				}
			}
			handlers++
		case OP_TRY_END:
			if handlers == 0 {
				return fmt.Errorf("handler stack underflow at %d", ip)
			}
			handlers--
		case OP_JUMP_IF_FALSE:
			need, delta = 1, -1
		case OP_RETURN:
			if handlers != 0 {
				return fmt.Errorf("return with active handler at %d", ip)
			}
			need = 1
		case OP_THROW:
			need = 1
		}
		if depth < need || depth+delta < 1 {
			return fmt.Errorf("stack underflow at %d", ip)
		}
		depth += delta
		switch op {
		case OP_THROW, OP_RETURN:
			continue
		case OP_JUMP:
			next += v
		case OP_LOOP:
			next -= v
		case OP_JUMP_IF_FALSE:
			if err := enqueue(next+v, stackState{depth, handlers}); err != nil {
				return err
			}
		}
		if err := enqueue(next, stackState{depth, handlers}); err != nil {
			return err
		}
	}
	return nil
}
