//go:build gen_code
// +build gen_code

package core

import (
	"io"
)

var TYPES = map[*string]*Type{}
var TYPE Types
var LINTER_TYPES = map[*string]bool{}

func init() {
	TYPE = Types{
		Associative:    RegInterface("Associative", (*Associative)(nil), ""),
		Callable:       RegInterface("Callable", (*Callable)(nil), ""),
		Collection:     RegInterface("Collection", (*Collection)(nil), ""),
		Comparable:     RegInterface("Comparable", (*Comparable)(nil), ""),
		Comparator:     RegInterface("Comparator", (*Comparator)(nil), ""),
		Counted:        RegInterface("Counted", (*Counted)(nil), ""),
		CountedIndexed: RegInterface("CountedIndexed", (*CountedIndexed)(nil), ""),
		Deref:          RegInterface("Deref", (*Deref)(nil), ""),
		Error:          RegInterface("Error", (*Error)(nil), ""),
		Gettable:       RegInterface("Gettable", (*Gettable)(nil), ""),
		Indexed:        RegInterface("Indexed", (*Indexed)(nil), ""),
		IOReader:       RegInterface("IOReader", (*io.Reader)(nil), ""),
		IOWriter:       RegInterface("IOWriter", (*io.Writer)(nil), ""),
		KVReduce:       RegInterface("KVReduce", (*KVReduce)(nil), ""),
		Reduce:         RegInterface("Reduce", (*Reduce)(nil), ""),
		Map:            RegInterface("Map", (*Map)(nil), ""),
		Meta:           RegInterface("Meta", (*Meta)(nil), ""),
		Named:          RegInterface("Named", (*Named)(nil), ""),
		Number:         RegInterface("Number", (*Number)(nil), ""),
		Pending:        RegInterface("Pending", (*Pending)(nil), ""),
		Ref:            RegInterface("Ref", (*Ref)(nil), ""),
		Reversible:     RegInterface("Reversible", (*Reversible)(nil), ""),
		Seq:            RegInterface("Seq", (*Seq)(nil), ""),
		Seqable:        RegInterface("Seqable", (*Seqable)(nil), ""),
		Sequential:     RegInterface("Sequential", (*Sequential)(nil), ""),
		Set:            RegInterface("Set", (*Set)(nil), ""),
		Stack:          RegInterface("Stack", (*Stack)(nil), ""),
		ArrayMap:       RegRefType("ArrayMap", (*ArrayMap)(nil), ""),
		ArrayMapSeq:    RegRefType("ArrayMapSeq", (*ArrayMapSeq)(nil), ""),
		ArrayNodeSeq:   RegRefType("ArrayNodeSeq", (*ArrayNodeSeq)(nil), ""),
		ArraySeq:       RegRefType("ArraySeq", (*ArraySeq)(nil), ""),
		MapSet:         RegRefType("MapSet", (*MapSet)(nil), ""),
		Atom:           RegRefType("Atom", (*Atom)(nil), ""),
		BigFloat:       RegRefType("BigFloat", (*BigFloat)(nil), "Wraps the Go 'math/big.Float' type"),
		BigInt:         RegRefType("BigInt", (*BigInt)(nil), "Wraps the Go 'math/big.Int' type"),
		Boolean:        RegType("Boolean", (*Boolean)(nil), "Wraps the Go 'bool' type"),
		Time:           RegType("Time", (*Time)(nil), "Wraps the Go 'time.Time' type"),
		Buffer:         RegRefType("Buffer", (*Buffer)(nil), ""),
		Char:           RegType("Char", (*Char)(nil), "Wraps the Go 'rune' type"),
		ConsSeq:        RegRefType("ConsSeq", (*ConsSeq)(nil), ""),
		Delay:          RegRefType("Delay", (*Delay)(nil), ""),
		Channel:        RegRefType("Channel", (*Channel)(nil), ""),
		Double:         RegType("Double", (*Double)(nil), "Wraps the Go 'float64' type"),
		EvalError:      RegRefType("EvalError", (*EvalError)(nil), ""),
		ExInfo:         RegRefType("ExInfo", (*ExInfo)(nil), ""),
		Fn:             RegRefType("Fn", (*Fn)(nil), "A callable function or macro implemented via Joker code"),
		File:           RegRefType("File", (*File)(nil), ""),
		BufferedReader: RegRefType("BufferedReader", (*BufferedReader)(nil), ""),
		HashMap:        RegRefType("HashMap", (*HashMap)(nil), ""),
		Int: RegType("Int", (*Int)(nil),
			"Wraps the Go 'int' type, which is 32 bits wide on 32-bit hosts, 64 bits wide on 64-bit hosts, etc."),
		Keyword:       RegType("Keyword", (*Keyword)(nil), "A possibly-namespace-qualified name prefixed by ':'"),
		LazySeq:       RegRefType("LazySeq", (*LazySeq)(nil), ""),
		List:          RegRefType("List", (*List)(nil), ""),
		MappingSeq:    RegRefType("MappingSeq", (*MappingSeq)(nil), ""),
		Namespace:     RegRefType("Namespace", (*Namespace)(nil), ""),
		Nil:           RegType("Nil", (*Nil)(nil), "The 'nil' value"),
		NodeSeq:       RegRefType("NodeSeq", (*NodeSeq)(nil), ""),
		ParseError:    RegRefType("ParseError", (*ParseError)(nil), ""),
		Proc:          RegRefType("Proc", (*Proc)(nil), "A callable function implemented via Go code"),
		Ratio:         RegRefType("Ratio", (*Ratio)(nil), "Wraps the Go 'math.big/Rat' type"),
		RecurBindings: RegRefType("RecurBindings", (*RecurBindings)(nil), ""),
		Regex:         RegRefType("Regex", (*Regex)(nil), "Wraps the Go 'regexp.Regexp' type"),
		String:        RegType("String", (*String)(nil), "Wraps the Go 'string' type"),
		Symbol:        RegType("Symbol", (*Symbol)(nil), ""),
		Type:          RegRefType("Type", (*Type)(nil), ""),
		Var:           RegRefType("Var", (*Var)(nil), ""),
		Vector:        RegRefType("Vector", (*Vector)(nil), ""),
		Vec:           RegInterface("Vec", (*Vec)(nil), ""),
		ArrayVector:   RegRefType("ArrayVector", (*ArrayVector)(nil), ""),
		VectorRSeq:    RegRefType("VectorRSeq", (*VectorRSeq)(nil), ""),
		VectorSeq:     RegRefType("VectorSeq", (*VectorSeq)(nil), ""),
	}
}
