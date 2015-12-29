package main

import (
	"fmt"
)

type (
	Expr interface {
		Eval() (Object, error)
	}
	LiteralExpr struct {
		obj Object
	}
	ParseError struct {
		obj ReadObject
		msg string
	}
)

func (err ParseError) Error() string {
	return fmt.Sprintf("stdin:%d:%d: %s", err.obj.line, err.obj.column, err.msg)
}

func (expr *LiteralExpr) Eval() (Object, error) {
	return expr.obj, nil
}

func parse(obj ReadObject) (Expr, error) {
	switch obj.obj.(type) {
	case Int, String, Char, Double, *BigInt, *BigFloat, Bool, Nil, *Ratio:
		return &LiteralExpr{obj: obj.obj}, nil
	default:
		return nil, &ParseError{obj: obj, msg: "Cannot parse form: " + obj.ToString(false)}
	}
}
