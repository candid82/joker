package core

import "maps"

// These facts describe the guarded integer path, not an assertion about mutable
// Vars. Every specialized call still verifies its implementation and operands.
func integerCallKind(e *CallExpr) Opcode {
	ref, ok := e.callable.(*VarRefExpr)
	if !ok || ref.vr == nil || ref.vr.ns != GLOBAL_ENV.CoreNamespace {
		return OP_CALL
	}
	switch ref.vr.name.Name() {
	case "inc":
		if len(e.args) == 1 {
			return OP_CALL_INT_INC
		}
	case "=":
		if len(e.args) == 2 {
			return OP_CALL_INT_EQ
		}
	}
	return OP_CALL
}

func integerExpr(e Expr, facts map[*Binding]bool) bool {
	switch e := e.(type) {
	case *LiteralExpr:
		_, ok := e.obj.(Int)
		return ok
	case *BindingExpr:
		return facts[e.binding]
	case *DoExpr:
		return len(e.body) > 0 && integerExpr(e.body[len(e.body)-1], facts)
	case *IfExpr:
		return integerExpr(e.positive, facts) && integerExpr(e.negative, facts)
	case *CallExpr:
		return integerCallKind(e) == OP_CALL_INT_INC && integerExpr(e.args[0], facts)
	case *LetExpr:
		if e.recursive {
			return false
		}
		inner := maps.Clone(facts)
		for i, value := range e.values {
			inner[e.bindings[i]] = integerExpr(value, inner)
		}
		return len(e.body) > 0 && integerExpr(e.body[len(e.body)-1], inner)
	}
	return false
}

// Join all backedges into the initial facts until they stop changing. Walkers
// stop at function bodies and nested loop bodies: their recur targets differ.
func (c *Compiler) inferIntegerLoop(e *LetExpr) {
	for changed := true; changed; {
		changed = false
		var walk func(Expr, map[*Binding]bool)
		walk = func(expr Expr, facts map[*Binding]bool) {
			walkBody := func(body []Expr, env map[*Binding]bool) {
				for _, x := range body {
					walk(x, env)
				}
			}
			switch x := expr.(type) {
			case *RecurExpr:
				for i, arg := range x.args {
					if c.integerBindings[e.bindings[i]] && !integerExpr(arg, facts) {
						c.integerBindings[e.bindings[i]] = false
						changed = true
					}
				}
			case *IfExpr:
				walk(x.cond, facts)
				walk(x.positive, facts)
				walk(x.negative, facts)
			case *DoExpr:
				walkBody(x.body, facts)
			case *LetExpr:
				inner := maps.Clone(facts)
				for i, value := range x.values {
					walk(value, inner)
					inner[x.bindings[i]] = !x.recursive && integerExpr(value, inner)
				}
				walkBody(x.body, inner)
			case *LoopExpr:
				walkBody(x.values, facts)
			case *CallExpr:
				walk(x.callable, facts)
				walkBody(x.args, facts)
			case *TryExpr:
				walkBody(x.body, facts)
				walkBody(x.finallyExpr, facts)
				for _, handler := range x.catches {
					walkBody(handler.body, facts)
				}
			case *MetaExpr:
				walk(x.meta, facts)
				walk(x.expr, facts)
			case *VectorExpr:
				walkBody(x.v, facts)
			case *MapExpr:
				walkBody(x.keys, facts)
				walkBody(x.values, facts)
			case *SetExpr:
				walkBody(x.elements, facts)
			case *ThrowExpr:
				walk(x.e, facts)
			case *DefExpr:
				if x.value != nil {
					walk(x.value, facts)
				}
			}
		}
		for _, body := range e.body {
			walk(body, c.integerBindings)
		}
	}
}

func (c *Compiler) integerCall(e *CallExpr) Opcode {
	if !c.integerOptimization {
		return OP_CALL
	}
	op := integerCallKind(e)
	if op != OP_CALL {
		for _, arg := range e.args {
			if !integerExpr(arg, c.integerBindings) {
				return OP_CALL
			}
		}
	}
	return op
}
