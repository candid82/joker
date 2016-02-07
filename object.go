package main

import (
	"fmt"
	"math/big"
	"strings"
)

type (
	Equality interface {
		Equals(interface{}) bool
	}
	Object interface {
		Equality
		ToString(escape bool) string
	}
	Meta interface {
		GetMeta() *ArrayMap
		WithMeta(*ArrayMap) Object
	}
	MetaHolder struct {
		meta *ArrayMap
	}
	Char     rune
	Double   float64
	Int      int
	BigInt   big.Int
	BigFloat big.Float
	Ratio    big.Rat
	Bool     bool
	Nil      struct{}
	Keyword  string
	Symbol   struct {
		MetaHolder
		ns   *string
		name *string
	}
	String string
	Regex  string
	Var    struct {
		MetaHolder
		ns    *Namespace
		name  Symbol
		value Object
	}
	Proc func([]Object) Object
	Fn   struct {
		fnExpr *FnExpr
		env    *Env
	}
)

func (fn *Fn) ToString(escape bool) string {
	return "function"
}

func (fn *Fn) Equals(other interface{}) bool {
	switch other := other.(type) {
	case *Fn:
		return fn == other
	default:
		return false
	}
}

func (fn *Fn) Call(args []Object) Object {
	return evalBody(fn.fnExpr.arities[0].body, fn.env)
}

func (p Proc) Call(args []Object) Object {
	return p(args)
}

func (p Proc) ToString(escape bool) string {
	return "primitive function"
}

func (p Proc) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Proc:
		return &p == &other
	default:
		return false
	}
}

func (m MetaHolder) GetMeta() *ArrayMap {
	return m.meta
}

func (sym Symbol) WithMeta(meta *ArrayMap) Object {
	res := sym
	res.meta = SafeMerge(res.meta, meta)
	return res
}

func (v *Var) WithMeta(meta *ArrayMap) Object {
	res := *v
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func MakeQualifiedSymbol(ns, name string) Symbol {
	return Symbol{
		ns:   STRINGS.Intern(ns),
		name: STRINGS.Intern(name),
	}
}

func MakeSymbol(nsname string) Symbol {
	index := strings.IndexRune(nsname, '/')
	if index == -1 || nsname == "/" {
		return Symbol{
			ns:   nil,
			name: STRINGS.Intern(nsname),
		}
	}
	return Symbol{
		ns:   STRINGS.Intern(nsname[0:index]),
		name: STRINGS.Intern(nsname[index+1 : len(nsname)]),
	}
}

func (v *Var) ToString(escape bool) string {
	return "#'" + v.ns.name.ToString(false) + "/" + v.name.ToString(false)
}

func (v *Var) Equals(other interface{}) bool {
	// TODO: revisit this
	return v == other
}

func (n Nil) ToString(escape bool) string {
	return "nil"
}

func (n Nil) Equals(other interface{}) bool {
	return n == other
}

func (rat *Ratio) ToString(escape bool) string {
	return (*big.Rat)(rat).String()
}

func (rat *Ratio) Equals(other interface{}) bool {
	if rat == other {
		return true
	}
	switch r := other.(type) {
	case *Ratio:
		return ((*big.Rat)(rat)).Cmp((*big.Rat)(r)) == 0
	case *BigInt:
		var otherRat big.Rat
		otherRat.SetInt((*big.Int)(r))
		return ((*big.Rat)(rat)).Cmp(&otherRat) == 0
	case Int:
		var otherRat big.Rat
		otherRat.SetInt64(int64(r))
		return ((*big.Rat)(rat)).Cmp(&otherRat) == 0
	}
	return false
}

func (bi *BigInt) ToString(escape bool) string {
	return (*big.Int)(bi).String() + "N"
}

func (bi *BigInt) Equals(other interface{}) bool {
	if bi == other {
		return true
	}
	switch b := other.(type) {
	case *BigInt:
		return ((*big.Int)(bi)).Cmp((*big.Int)(b)) == 0
	case Int:
		bi2 := big.NewInt(int64(b))
		return ((*big.Int)(bi)).Cmp(bi2) == 0
	}
	return false
}

func (bf *BigFloat) ToString(escape bool) string {
	return (*big.Float)(bf).Text('g', 256) + "M"
}

func (bf *BigFloat) Equals(other interface{}) bool {
	if bf == other {
		return true
	}
	switch b := other.(type) {
	case *BigFloat:
		return ((*big.Float)(bf)).Cmp((*big.Float)(b)) == 0
	case Double:
		bf2 := big.NewFloat(float64(b))
		return ((*big.Float)(bf)).Cmp(bf2) == 0
	}
	return false
}

func (c Char) ToString(escape bool) string {
	if escape {
		return escapeRune(rune(c))
	}
	return string(c)
}

func (c Char) Equals(other interface{}) bool {
	return c == other
}

func (d Double) ToString(escape bool) string {
	return fmt.Sprintf("%f", float64(d))
}

func (d Double) Equals(other interface{}) bool {
	return d == other
}

func (i Int) ToString(escape bool) string {
	return fmt.Sprintf("%d", int(i))
}

func (i Int) Equals(other interface{}) bool {
	return i == other
}

func (b Bool) ToString(escape bool) string {
	return fmt.Sprintf("%t", bool(b))
}

func (b Bool) Equals(other interface{}) bool {
	return b == other
}

func (k Keyword) ToString(escape bool) string {
	return string(k)
}

func (k Keyword) Equals(other interface{}) bool {
	return k == other
}

func (rx Regex) ToString(escape bool) string {
	if escape {
		return "#" + escapeString(string(rx))
	}
	return "#" + string(rx)
}

func (rx Regex) Equals(other interface{}) bool {
	return rx == other
}

func (s Symbol) ToString(escape bool) string {
	if s.ns != nil {
		return *s.ns + "/" + *s.name
	}
	return *s.name
}

func (s Symbol) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Symbol:
		return s.ns == other.ns && s.name == other.name
	default:
		return false
	}
}

func (s String) ToString(escape bool) string {
	if escape {
		return escapeString(string(s))
	}
	return string(s)
}

func (s String) Equals(other interface{}) bool {
	return s == other
}

func IsSymbol(obj Object) bool {
	switch obj.(type) {
	case Symbol:
		return true
	default:
		return false
	}
}

func IsVector(obj Object) bool {
	switch obj.(type) {
	case *Vector:
		return true
	default:
		return false
	}
}

func IsSeq(obj Object) bool {
	switch obj.(type) {
	case Seq:
		return true
	default:
		return false
	}
}
