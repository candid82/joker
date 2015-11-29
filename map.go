package main

import (
	"bytes"
)

type (
	ArrayMap struct {
		arr []Object
	}
)

func (m *ArrayMap) indexOf(key Object) int {
	for i := 0; i < len(m.arr); i += 2 {
		if m.arr[i].Equal(key) {
			return i
		}
	}
	return -1
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

func (m *ArrayMap) Equal(other interface{}) bool {
	switch otherMap := other.(type) {
	case *ArrayMap:
		if m == otherMap {
			return true
		}
		if len(m.arr) != len(otherMap.arr) {
			return false
		}
		for i := 0; i < len(m.arr); i += 2 {
			success, value := otherMap.Get(m.arr[i])
			if !success || value != m.arr[i+1] {
				return false
			}
		}
		for i := 0; i < len(otherMap.arr); i += 2 {
			success, value := m.Get(otherMap.arr[i])
			if !success || value != otherMap.arr[i+1] {
				return false
			}
		}
		return true
	default:
		return false
	}
}
