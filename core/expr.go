package core

func dumpPosition(p Position) Map {
	res := EmptyArrayMap()
	res.Add(KEYWORDS.startLine, Int{I: p.startLine})
	res.Add(KEYWORDS.endLine, Int{I: p.endLine})
	res.Add(KEYWORDS.startColumn, Int{I: p.startColumn})
	res.Add(KEYWORDS.endColumn, Int{I: p.endColumn})
	res.Add(KEYWORDS.filename, String{S: p.Filename()})
	return res
}

func exprArrayMap(expr Expr, exprType string, pos bool) *ArrayMap {
	res := EmptyArrayMap()
	res.Add(KEYWORDS.type_, MakeKeyword(exprType))
	if pos {
		res.Add(KEYWORDS.pos, dumpPosition(expr.Pos()))
	}
	return res
}

func addVector(res *ArrayMap, body []Expr, name string, pos bool) {
	b := EmptyVector()
	for _, e := range body {
		b = b.Conjoin(e.Dump(pos))
	}
	res.Add(MakeKeyword(name), b)
}

func (expr *LiteralExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "literal", pos)
	res.Add(KEYWORDS.object, expr.obj)
	return res
}

func (expr *VectorExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "vector", pos)
	addVector(res, expr.v, "vector", pos)
	return res
}

func (expr *MapExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "map", pos)
	addVector(res, expr.keys, "keys", pos)
	addVector(res, expr.values, "values", pos)
	return res
}

func (expr *SetExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "set", pos)
	addVector(res, expr.elements, "set", pos)
	return res
}

func (expr *IfExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "if", pos)
	res.Add(MakeKeyword("condition"), expr.cond.Dump(pos))
	res.Add(MakeKeyword("positive"), expr.positive.Dump(pos))
	res.Add(MakeKeyword("negative"), expr.negative.Dump(pos))
	return res
}

func (expr *DefExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "def", pos)
	res.Add(KEYWORDS.var_, expr.vr)
	res.Add(KEYWORDS.name, expr.name)
	if expr.value != nil {
		res.Add(KEYWORDS.value, expr.value.Dump(pos))
	}
	if expr.meta != nil {
		res.Add(KEYWORDS.meta, expr.meta.Dump(pos))
	}
	return res
}

func (expr *CallExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "call", pos)
	res.Add(MakeKeyword("name"), String{S: expr.Name()})
	res.Add(MakeKeyword("callable"), expr.callable.Dump(pos))
	addVector(res, expr.args, "args", pos)
	return res
}

func (expr *MacroCallExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "macro-call", pos)
	res.Add(MakeKeyword("name"), String{S: expr.name})
	args := EmptyVector()
	for _, arg := range expr.args {
		args = args.Conjoin(arg)
	}
	res.Add(MakeKeyword("args"), args)
	return res
}

func (expr *RecurExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "recur", pos)
	addVector(res, expr.args, "args", pos)
	return res
}

func (expr *VarRefExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "var-ref", pos)
	res.Add(KEYWORDS.var_, expr.vr)
	return res
}

func (expr *SetMacroExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "set-macro", pos)
	res.Add(KEYWORDS.var_, expr.vr)
	return res
}

func (expr *BindingExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "binding", pos)
	res.Add(MakeKeyword("name"), expr.binding.name)
	return res
}

func (expr *MetaExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "meta", pos)
	res.Add(KEYWORDS.meta, expr.meta.Dump(pos))
	res.Add(MakeKeyword("expr"), expr.expr.Dump(pos))
	return res
}

func (expr *DoExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "do", pos)
	addVector(res, expr.body, "body", pos)
	return res
}

func (expr *FnArityExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "arity", pos)
	args := EmptyVector()
	for _, arg := range expr.args {
		args = args.Conjoin(arg)
	}
	res.Add(MakeKeyword("args"), args)
	addVector(res, expr.body, "body", pos)
	return res
}

func (expr *FnExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "fn", pos)
	if expr.self.name != nil {
		res.Add(MakeKeyword("self"), expr.self)
	}
	if expr.variadic != nil {
		res.Add(MakeKeyword("variadic"), expr.variadic.Dump(pos))
	}
	arities := EmptyVector()
	for _, a := range expr.arities {
		arities = arities.Conjoin(a.Dump(pos))
	}
	res.Add(MakeKeyword("arities"), arities)
	return res
}

func (expr *LetExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "let", pos)
	names := EmptyVector()
	for _, name := range expr.names {
		names = names.Conjoin(name)
	}
	addVector(res, expr.values, "values", pos)
	addVector(res, expr.body, "body", pos)
	return res
}

func (expr *LoopExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "loop", pos)
	names := EmptyVector()
	for _, name := range expr.names {
		names = names.Conjoin(name)
	}
	addVector(res, expr.values, "values", pos)
	addVector(res, expr.body, "body", pos)
	return res
}

func (expr *ThrowExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "throw", pos)
	res.Add(MakeKeyword("expr"), expr.e.Dump(pos))
	return res
}

func (expr *CatchExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "catch", pos)
	res.Add(MakeKeyword("error-type"), expr.excType)
	res.Add(MakeKeyword("error-symbol"), expr.excSymbol)
	addVector(res, expr.body, "body", pos)
	return res
}

func (expr *TryExpr) Dump(pos bool) Map {
	res := exprArrayMap(expr, "try", pos)
	addVector(res, expr.body, "body", pos)
	addVector(res, expr.finallyExpr, "finally", pos)
	catches := EmptyVector()
	for _, c := range expr.catches {
		catches = catches.Conjoin(c.Dump(pos))
	}
	res.Add(MakeKeyword("catches"), catches)
	return res
}
