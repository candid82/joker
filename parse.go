package main

type (
	Expr interface {
		Eval() (Object, error)
	}
	LiteralExpr struct {
		obj Object
	}
	ParseError struct {
		obj Object
		msg string
	}
)

func (err *ParseError) Error() string {
	return err.msg
}

func (expr *LiteralExpr) Eval() (Object, error) {
	return expr.obj, nil
}

func parse(obj Object) (Expr, error) {
	switch obj.(type) {
	case Int, String, Char, Double, *BigInt, *BigFloat, Bool, Nil, *Ratio:
		return &LiteralExpr{obj: obj}, nil
	default:
		return nil, &ParseError{obj: obj, msg: "Cannot parse form: " + obj.ToString(false)}
	}
}
