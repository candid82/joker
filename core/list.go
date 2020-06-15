package core

import "io"

type List struct {
	InfoHolder
	MetaHolder
	first Object
	rest  *List
	count int
}

func NewList(first Object, rest *List) *List {
	result := List{
		first: first,
		rest:  rest,
	}
	if rest != nil {
		result.count = rest.count + 1
	}
	return &result
}

func NewListFrom(objs ...Object) *List {
	res := EmptyList
	for i := len(objs) - 1; i >= 0; i-- {
		res = res.conj(objs[i])
	}
	return res
}

func (list *List) WithMeta(meta Map) Object {
	res := *list
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (list *List) conj(obj Object) *List {
	return NewList(obj, list)
}

func (list *List) Conj(obj Object) Conjable {
	return list.conj(obj)
}

func (list *List) ToString(escape bool) string {
	return SeqToString(list, escape)
}

func (seq *List) Pprint(w io.Writer, indent int) int {
	return pprintSeq(seq, w, indent)
}

func (seq *List) Format(w io.Writer, indent int) int {
	return formatSeq(seq, w, indent)
}

func (list *List) Equals(other interface{}) bool {
	return IsSeqEqual(list, other)
}

func (list *List) GetType() *Type {
	return TYPE.List
}

func (list *List) Hash() uint32 {
	return hashOrdered(list)
}

func (list *List) First() Object {
	return list.first
}

func (list *List) Rest() Seq {
	return list.rest
}

func (list *List) IsEmpty() bool {
	return list.count == 0
}

func (list *List) Cons(obj Object) Seq {
	return list.conj(obj)
}

func (list *List) Seq() Seq {
	return list
}

func (list *List) Second() Object {
	return list.rest.first
}

func (list *List) Third() Object {
	return list.rest.rest.first
}

func (list *List) Forth() Object {
	return list.rest.rest.rest.first
}

func (list *List) Count() int {
	return list.count
}

func (list *List) Empty() Collection {
	return EmptyList
}

func (list *List) Peek() Object {
	return list.first
}

func (list *List) Pop() Stack {
	if list.count == 0 {
		panic(RT.NewError("Can't pop empty list"))
	}
	return list.rest
}

func (list *List) sequential() {}

var EmptyList = NewList(Nil{}, nil)

func init() {
	EmptyList.rest = EmptyList
}
