package core

func (expr *LiteralExpr) InferType() *Type {
	return expr.obj.GetType()
}

func (expr *VectorExpr) InferType() *Type {
	return TYPE.Vector
}

func (expr *MapExpr) InferType() *Type {
	return TYPE.ArrayMap
}

func (expr *SetExpr) InferType() *Type {
	return TYPE.MapSet
}

func (expr *IfExpr) InferType() *Type {
	return nil
}

func (expr *DefExpr) InferType() *Type {
	return TYPE.Var
}

func (expr *CallExpr) InferType() *Type {
	return nil
}

func (expr *MacroCallExpr) InferType() *Type {
	return nil
}

func (expr *RecurExpr) InferType() *Type {
	return nil
}

func (expr *VarRefExpr) InferType() *Type {
	// if expr.vr.taggedType != nil {
	// 	return expr.vr.taggedType
	// }
	if expr.vr.expr != nil {
		return expr.vr.expr.InferType()
	}
	return nil
}

func (expr *BindingExpr) InferType() *Type {
	return nil
}

func (expr *MetaExpr) InferType() *Type {
	return expr.expr.InferType()
}

func typeOfLast(exprs []Expr) *Type {
	n := len(exprs)
	if n > 0 {
		return exprs[n-1].InferType()
	}
	return nil
}

func (expr *DoExpr) InferType() *Type {
	return typeOfLast(expr.body)
}

func (expr *FnExpr) InferType() *Type {
	return TYPE.Fn
}

func (expr *LetExpr) InferType() *Type {
	return typeOfLast(expr.body)
}

func (expr *LoopExpr) InferType() *Type {
	return typeOfLast(expr.body)
}

func (expr *ThrowExpr) InferType() *Type {
	return nil
}

func (expr *CatchExpr) InferType() *Type {
	return typeOfLast(expr.body)
}

func (expr *TryExpr) InferType() *Type {
	return typeOfLast(expr.body)
}
