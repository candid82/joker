package core

import (
	"bytes"
	"fmt"
	"io"
)

type (
	Map interface {
		Associative
		Seqable
		Counted
		Without(key Object) Map
		Keys() Seq
		Vals() Seq
		Merge(m Map) Map
		Iter() MapIterator
	}
	MapIterator interface {
		HasNext() bool
		Next() *Pair
	}
	EmptyMapIterator struct {
	}
	Pair struct {
		Key   Object
		Value Object
	}
)

var (
	emptyMapIterator = &EmptyMapIterator{}
)

func (iter *EmptyMapIterator) HasNext() bool {
	return false
}

func (iter *EmptyMapIterator) Next() *Pair {
	panic(newIteratorError())
}

func mapConj(m Map, obj Object) Conjable {
	switch obj := obj.(type) {
	case *Vector:
		if obj.count != 2 {
			panic(RT.NewError("Vector argument to map's conj must be a vector with two elements"))
		}
		return m.Assoc(obj.at(0), obj.at(1))
	case Map:
		return m.Merge(obj)
	default:
		panic(RT.NewError("Argument to map's conj must be a vector with two elements or a map"))
	}
}

func mapEquals(m Map, other interface{}) bool {
	if m == other {
		return true
	}
	switch otherMap := other.(type) {
	case Nil:
		return false
	case Map:
		if m.Count() != otherMap.Count() {
			return false
		}
		for iter := m.Iter(); iter.HasNext(); {
			p := iter.Next()
			success, value := otherMap.Get(p.Key)
			if !success || !value.Equals(p.Value) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func mapToString(m Map, escape bool) string {
	var b bytes.Buffer
	b.WriteRune('{')
	if m.Count() > 0 {
		for iter := m.Iter(); ; {
			p := iter.Next()
			b.WriteString(p.Key.ToString(escape))
			b.WriteRune(' ')
			b.WriteString(p.Value.ToString(escape))
			if iter.HasNext() {
				b.WriteString(", ")
			} else {
				break
			}
		}
	}
	b.WriteRune('}')
	return b.String()
}

func callMap(m Map, args []Object) Object {
	CheckArity(args, 1, 2)
	if ok, v := m.Get(args[0]); ok {
		return v
	}
	if len(args) == 2 {
		return args[1]
	}
	return NIL
}

func pprintMap(m Map, w io.Writer, indent int) int {
	i := indent + 1
	fmt.Fprint(w, "{")
	if m.Count() > 0 {
		for iter := m.Iter(); ; {
			p := iter.Next()
			i = pprintObject(p.Key, indent+1, w)
			fmt.Fprint(w, " ")
			i = pprintObject(p.Value, i+1, w)
			if iter.HasNext() {
				fmt.Fprint(w, "\n")
				writeIndent(w, indent+1)
			} else {
				break
			}
		}
	}
	fmt.Fprint(w, "}")
	return i + 1
}
