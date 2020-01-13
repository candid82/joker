// +build !fast_init

package core

import (
	"io"
	"reflect"
)

var TYPES = map[*string]*Type{}
var TYPE Types

func RegRefType(name string, inst interface{}, doc string) *Type {
	if doc != "" {
		doc = "\n  " + doc
	}
	meta := MakeMeta(nil, "(Concrete reference type)"+doc, "1.0")
	meta.Add(KEYWORDS.name, MakeString(name))
	t := &Type{MetaHolder{meta}, name, reflect.TypeOf(inst)}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func regType(name string, inst interface{}, doc string) *Type {
	if doc != "" {
		doc = "\n  " + doc
	}
	meta := MakeMeta(nil, "(Concrete type)"+doc, "1.0")
	meta.Add(KEYWORDS.name, MakeString(name))
	t := &Type{MetaHolder{meta}, name, reflect.TypeOf(inst).Elem()}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func regInterface(name string, inst interface{}, doc string) *Type {
	if doc != "" {
		doc = "\n  " + doc
	}
	meta := MakeMeta(nil, "(Interface type)"+doc, "1.0")
	meta.Add(KEYWORDS.name, MakeString(name))
	t := &Type{MetaHolder{meta}, name, reflect.TypeOf(inst).Elem()}
	TYPES[STRINGS.Intern(name)] = t
	return t
}

func init() {
	TYPE = Types{
		Associative:    regInterface("Associative", (*Associative)(nil), ""),
		Callable:       regInterface("Callable", (*Callable)(nil), ""),
		Collection:     regInterface("Collection", (*Collection)(nil), ""),
		Comparable:     regInterface("Comparable", (*Comparable)(nil), ""),
		Comparator:     regInterface("Comparator", (*Comparator)(nil), ""),
		Counted:        regInterface("Counted", (*Counted)(nil), ""),
		Deref:          regInterface("Deref", (*Deref)(nil), ""),
		Error:          regInterface("Error", (*Error)(nil), ""),
		Gettable:       regInterface("Gettable", (*Gettable)(nil), ""),
		Indexed:        regInterface("Indexed", (*Indexed)(nil), ""),
		IOReader:       regInterface("IOReader", (*io.Reader)(nil), ""),
		IOWriter:       regInterface("IOWriter", (*io.Writer)(nil), ""),
		KVReduce:       regInterface("KVReduce", (*KVReduce)(nil), ""),
		Map:            regInterface("Map", (*Map)(nil), ""),
		Meta:           regInterface("Meta", (*Meta)(nil), ""),
		Named:          regInterface("Named", (*Named)(nil), ""),
		Number:         regInterface("Number", (*Number)(nil), ""),
		Pending:        regInterface("Pending", (*Pending)(nil), ""),
		Ref:            regInterface("Ref", (*Ref)(nil), ""),
		Reversible:     regInterface("Reversible", (*Reversible)(nil), ""),
		Seq:            regInterface("Seq", (*Seq)(nil), ""),
		Seqable:        regInterface("Seqable", (*Seqable)(nil), ""),
		Sequential:     regInterface("Sequential", (*Sequential)(nil), ""),
		Set:            regInterface("Set", (*Set)(nil), ""),
		Stack:          regInterface("Stack", (*Stack)(nil), ""),
		ArrayMap:       RegRefType("ArrayMap", (*ArrayMap)(nil), ""),
		ArrayMapSeq:    RegRefType("ArrayMapSeq", (*ArrayMapSeq)(nil), ""),
		ArrayNodeSeq:   RegRefType("ArrayNodeSeq", (*ArrayNodeSeq)(nil), ""),
		ArraySeq:       RegRefType("ArraySeq", (*ArraySeq)(nil), ""),
		MapSet:         RegRefType("MapSet", (*MapSet)(nil), ""),
		Atom:           RegRefType("Atom", (*Atom)(nil), ""),
		BigFloat:       RegRefType("BigFloat", (*BigFloat)(nil), ""),
		BigInt:         RegRefType("BigInt", (*BigInt)(nil), ""),
		Boolean:        regType("Boolean", (*Boolean)(nil), ""),
		Time:           regType("Time", (*Time)(nil), ""),
		Buffer:         RegRefType("Buffer", (*Buffer)(nil), ""),
		Char:           regType("Char", (*Char)(nil), ""),
		ConsSeq:        RegRefType("ConsSeq", (*ConsSeq)(nil), ""),
		Delay:          RegRefType("Delay", (*Delay)(nil), ""),
		Channel:        RegRefType("Channel", (*Channel)(nil), ""),
		Double:         regType("Double", (*Double)(nil), ""),
		EvalError:      RegRefType("EvalError", (*EvalError)(nil), ""),
		ExInfo:         RegRefType("ExInfo", (*ExInfo)(nil), ""),
		Fn:             RegRefType("Fn", (*Fn)(nil), ""),
		File:           RegRefType("File", (*File)(nil), ""),
		BufferedReader: RegRefType("BufferedReader", (*BufferedReader)(nil), ""),
		HashMap:        RegRefType("HashMap", (*HashMap)(nil), ""),
		Int: regType("Int", (*Int)(nil),
			"Wraps the Go 'int' type, which is 32 bits wide on 32-bit hosts, 64 bits wide on 64-bit hosts, etc."),
		Keyword:       regType("Keyword", (*Keyword)(nil), ""),
		LazySeq:       RegRefType("LazySeq", (*LazySeq)(nil), ""),
		List:          RegRefType("List", (*List)(nil), ""),
		MappingSeq:    RegRefType("MappingSeq", (*MappingSeq)(nil), ""),
		Namespace:     RegRefType("Namespace", (*Namespace)(nil), ""),
		Nil:           regType("Nil", (*Nil)(nil), ""),
		NodeSeq:       RegRefType("NodeSeq", (*NodeSeq)(nil), ""),
		ParseError:    RegRefType("ParseError", (*ParseError)(nil), ""),
		Proc:          RegRefType("Proc", (*Proc)(nil), ""),
		ProcFn:        RegRefType("ProcFn", (*ProcFn)(nil), ""),
		Ratio:         RegRefType("Ratio", (*Ratio)(nil), ""),
		RecurBindings: RegRefType("RecurBindings", (*RecurBindings)(nil), ""),
		Regex:         RegRefType("Regex", (*Regex)(nil), ""),
		String:        regType("String", (*String)(nil), ""),
		Symbol:        regType("Symbol", (*Symbol)(nil), ""),
		Type:          RegRefType("Type", (*Type)(nil), ""),
		Var:           RegRefType("Var", (*Var)(nil), ""),
		Vector:        RegRefType("Vector", (*Vector)(nil), ""),
		VectorRSeq:    RegRefType("VectorRSeq", (*VectorRSeq)(nil), ""),
		VectorSeq:     RegRefType("VectorSeq", (*VectorSeq)(nil), ""),
	}
}
