package main

import (
	"bytes"
)

type List struct {
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

func (list *List) Cons(obj Object) *List {
	return NewList(obj, list)
}

func (list *List) ToString(escape bool) string {
	var b bytes.Buffer
	b.WriteRune('(')
	for list.count > 0 {
		b.WriteString(list.first.ToString(escape))
		list = list.rest
		if list.count > 0 {
			b.WriteRune(' ')
		}
	}
	b.WriteRune(')')
	return b.String()
}

func (list *List) Equals(other interface{}) bool {
	switch otherList := other.(type) {
	case *List:
		if list == otherList {
			return true
		}
		if list.count != otherList.count {
			return false
		}
		for list.count > 0 {
			if !list.first.Equals(otherList.first) {
				return false
			}
			list = list.rest
			otherList = otherList.rest
		}
		return true
	default:
		return false
	}
}

var EmptyList = NewList(nil, nil)
