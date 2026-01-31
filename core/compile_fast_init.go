//go:build !gen_code
// +build !gen_code

package core

// CompileAST walks the expression tree and pre-compiles VM-compatible FnExpr nodes.
// This should be called after parsing but before evaluation, when all macros have
// been expanded.
func CompileAST(expr Expr) {
	if DISABLE_VM {
		return
	}
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *FnExpr:
		// Try to compile this function if it's VM-compatible
		if IsVMCompatibleFn(e) {
			if proto, err := tryCompileFnExpr(e); err == nil {
				e.compiled = proto
			}
		}
		// Walk the function bodies recursively
		for _, arity := range e.arities {
			for _, bodyExpr := range arity.body {
				CompileAST(bodyExpr)
			}
		}
		if e.variadic != nil {
			for _, bodyExpr := range e.variadic.body {
				CompileAST(bodyExpr)
			}
		}
	case *IfExpr:
		CompileAST(e.cond)
		CompileAST(e.positive)
		CompileAST(e.negative)
	case *LetExpr:
		for _, v := range e.values {
			CompileAST(v)
		}
		for _, b := range e.body {
			CompileAST(b)
		}
	case *LoopExpr:
		for _, v := range e.values {
			CompileAST(v)
		}
		for _, b := range e.body {
			CompileAST(b)
		}
	case *DoExpr:
		for _, b := range e.body {
			CompileAST(b)
		}
	case *DefExpr:
		if e.value != nil {
			CompileAST(e.value)
		}
		if e.meta != nil {
			CompileAST(e.meta)
		}
	case *CallExpr:
		CompileAST(e.callable)
		for _, arg := range e.args {
			CompileAST(arg)
		}
	case *VectorExpr:
		for _, elem := range e.v {
			CompileAST(elem)
		}
	case *MapExpr:
		for i := range e.keys {
			CompileAST(e.keys[i])
			CompileAST(e.values[i])
		}
	case *SetExpr:
		for _, elem := range e.elements {
			CompileAST(elem)
		}
	case *MetaExpr:
		CompileAST(e.meta)
		CompileAST(e.expr)
	case *TryExpr:
		for _, b := range e.body {
			CompileAST(b)
		}
		for _, c := range e.catches {
			for _, b := range c.body {
				CompileAST(b)
			}
		}
		for _, f := range e.finallyExpr {
			CompileAST(f)
		}
	case *ThrowExpr:
		CompileAST(e.e)
	case *RecurExpr:
		for _, arg := range e.args {
			CompileAST(arg)
		}
	// These don't contain nested expressions that need compilation:
	case *LiteralExpr, *VarRefExpr, *BindingExpr, *SetMacroExpr, *MacroCallExpr:
		// Nothing to do
	}
}
