package main

import (
	"bytes"
)

type (
	ArrayMap struct {
		arr []Object
	}
	ArrayMapIterator struct {
		m       *ArrayMap
		current int
	}
	Pair struct {
		key   Object
		value Object
	}
)

func (iter *ArrayMapIterator) Next() Pair {
	res := Pair{
		key:   iter.m.arr[iter.current],
		value: iter.m.arr[iter.current+1],
	}
	iter.current += 2
	return res
}

func (iter *ArrayMapIterator) HasNext() bool {
	return iter.current < len(iter.m.arr)
}

func (m *ArrayMap) indexOf(key Object) int {
	for i := 0; i < len(m.arr); i += 2 {
		if m.arr[i].Equals(key) {
			return i
		}
	}
	return -1
}

func ArraySeqFromArrayMap(m *ArrayMap) *ArraySeq {
	return &ArraySeq{arr: m.arr}
}

func (m *ArrayMap) Get(key Object) (bool, Object) {
	i := m.indexOf(key)
	if i != -1 {
		return true, m.arr[i+1]
	}
	return false, nil
}

func (m *ArrayMap) Set(key Object, value Object) {
	i := m.indexOf(key)
	if i != -1 {
		m.arr[i+1] = value
	} else {
		m.arr = append(m.arr, key)
		m.arr = append(m.arr, value)
	}
}

func (m *ArrayMap) Add(key Object, value Object) bool {
	i := m.indexOf(key)
	if i != -1 {
		return false
	}
	m.arr = append(m.arr, key)
	m.arr = append(m.arr, value)
	return true
}

func (m *ArrayMap) Count() int {
	return len(m.arr) / 2
}

func (m *ArrayMap) Assoc(key Object, value Object) *ArrayMap {
	result := ArrayMap{arr: make([]Object, len(m.arr), cap(m.arr))}
	copy(result.arr, m.arr)
	result.Set(key, value)
	return &result
}

func (m *ArrayMap) Without(key Object) *ArrayMap {
	result := ArrayMap{arr: make([]Object, len(m.arr), cap(m.arr))}
	var i, j int
	for i, j = 0, 0; i < len(m.arr); i += 2 {
		if m.arr[i].Equals(key) {
			continue
		}
		result.arr[j] = m.arr[i]
		result.arr[j+1] = m.arr[i+1]
		j += 2
	}
	if i != j {
		result.arr = result.arr[:j]
	}
	return &result
}

func (m *ArrayMap) Keys() Seq {
	mlen := len(m.arr) / 2
	res := make([]Object, mlen)
	for i := 0; i < mlen; i++ {
		res[i] = m.arr[i*2]
	}
	return &ArraySeq{arr: res}
}

func (m *ArrayMap) iter() *ArrayMapIterator {
	return &ArrayMapIterator{m: m}
}

func EmptyArrayMap() *ArrayMap {
	return &ArrayMap{}
}

func (m *ArrayMap) ToString(escape bool) string {
	var b bytes.Buffer
	b.WriteRune('{')
	if len(m.arr) > 0 {
		for i := 0; i < len(m.arr)-2; i += 2 {
			b.WriteString(m.arr[i].ToString(escape))
			b.WriteRune(' ')
			b.WriteString(m.arr[i+1].ToString(escape))
			b.WriteString(", ")
		}
		b.WriteString(m.arr[len(m.arr)-2].ToString(escape))
		b.WriteRune(' ')
		b.WriteString(m.arr[len(m.arr)-1].ToString(escape))
	}
	b.WriteRune('}')
	return b.String()
}

func (m *ArrayMap) Equals(other interface{}) bool {
	if m == other {
		return true
	}
	switch otherMap := other.(type) {
	case *ArrayMap:
		if len(m.arr) != len(otherMap.arr) {
			return false
		}
		for i := 0; i < len(m.arr); i += 2 {
			success, value := otherMap.Get(m.arr[i])
			if !success || !value.Equals(m.arr[i+1]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
