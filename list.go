package main

type List struct {
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
		res = res.Conj(objs[i])
	}
	return res
}

func (list *List) WithMeta(meta *ArrayMap) Object {
	res := *list
	res.meta = meta
	return &res
}

func (list *List) Conj(obj Object) *List {
	return NewList(obj, list)
}

func (list *List) ToString(escape bool) string {
	return SeqToString(list, escape)
}

func (list *List) Equals(other interface{}) bool {
	if list == other {
		return true
	}
	switch s := other.(type) {
	case Sequenceable:
		return SeqsEqual(list, s.Seq())
	default:
		return false
	}
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
	return list.Conj(obj)
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

var EmptyList = NewList(Nil{}, nil)

func init() {
	EmptyList.rest = EmptyList
}
