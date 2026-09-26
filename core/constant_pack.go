package core

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
)

const (
	constantRef byte = iota
	constantDef
	constantNil
	constantBool
	constantInt
	constantDouble
	constantString
	constantChar
	constantKeyword
	constantSymbol
	constantVar
	constantType
	constantNamespace
	constantBigInt
	constantBigFloat
	constantRatio
	constantRegex
	constantList
	constantArrayVector
	constantVector
	constantArrayMap
	constantHashMap
	constantSet
	constantVectorSeq
	constantVectorRSeq
	constantArraySeq
	constantConsSeq
	constantArrayMapSeq
)

func packBytes(p, b []byte) []byte { p = appendInt(p, len(b)); return append(p, b...) }
func unpackBytes(p []byte) ([]byte, []byte) {
	n, p := extractInt(p)
	if n < 0 || n > len(p) {
		panic(RT.NewError("Invalid packed byte string"))
	}
	return p[:n], p[n:]
}

// Runtime constant pools accept every Object. Persistence deliberately accepts
// only portable data; unsupported host/runtime values fail here, not in Compile.
// Pointer references preserve sharing across nested constants and prototypes.
func packObject(obj Object, p []byte, env *PackEnv) []byte {
	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		if env.objects == nil {
			env.objects = make(map[Object]int)
			env.packing = make(map[Object]bool)
		}
		if id, ok := env.objects[obj]; ok {
			if env.packing[obj] {
				panic(RT.NewError("Cyclic constant cannot be packed"))
			}
			return appendInt(append(p, constantRef), id)
		}
		env.objects[obj] = len(env.objects)
		env.packing[obj] = true
		defer delete(env.packing, obj)
		p = append(p, constantDef)
	}
	switch v := obj.(type) {
	case Nil:
		p = append(p, constantNil)
	case Boolean:
		p = appendBool(append(p, constantBool), v.B)
	case Int:
		p = appendInt(append(p, constantInt), v.I)
	case Double:
		p = appendInt(append(p, constantDouble), int(math.Float64bits(v.D)))
	case String:
		p = packBytes(append(p, constantString), []byte(v.S))
	case Char:
		p = appendInt(append(p, constantChar), int(v.Ch))
	case Keyword:
		p = packBytes(append(p, constantKeyword), []byte(v.ToString(false)[1:]))
	case Symbol:
		return v.Pack(append(p, constantSymbol), env)
	case *Var:
		return v.Pack(append(p, constantVar), env)
	case *Type:
		return v.Pack(append(p, constantType), env)
	case *Namespace:
		return v.Name.Pack(append(p, constantNamespace), env)
	case *BigInt:
		data, err := v.b.GobEncode()
		PanicOnErr(err)
		p = packBytes(append(p, constantBigInt), data)
	case *BigFloat:
		data, err := v.b.GobEncode()
		PanicOnErr(err)
		p = packBytes(append(p, constantBigFloat), data)
	case *Ratio:
		data, err := v.r.GobEncode()
		PanicOnErr(err)
		p = packBytes(append(p, constantRatio), data)
	case *Regex:
		p = packBytes(append(p, constantRegex), []byte(v.R.String()))
	case *List:
		p = appendBool(append(p, constantList), v.IsEmpty())
		if !v.IsEmpty() {
			p = packObject(v.first, p, env)
			p = packObject(v.rest, p, env)
		}
	case *VectorSeq:
		p = appendInt(append(p, constantVectorSeq), v.index)
		p = packObject(v.vector.(Object), p, env)
	case *VectorRSeq:
		p = appendInt(append(p, constantVectorRSeq), v.index)
		p = packObject(v.vector.(Object), p, env)
	case *ArraySeq:
		p = appendInt(append(p, constantArraySeq), v.index)
		p = appendInt(p, v.step)
		p = appendInt(p, len(v.arr))
		for _, el := range v.arr {
			p = packObject(el, p, env)
		}
	case *ConsSeq:
		p = append(p, constantConsSeq)
		p = packObject(v.first, p, env)
		p = packObject(v.rest, p, env)
	case *ArrayMapSeq:
		p = appendInt(append(p, constantArrayMapSeq), v.index)
		p = packObject(v.m, p, env)
	case Vec:
		tag := constantVector
		if _, ok := v.(*ArrayVector); ok {
			tag = constantArrayVector
		}
		p = appendInt(append(p, tag), v.Count())
		for i := 0; i < v.Count(); i++ {
			p = packObject(v.At(i), p, env)
		}
	case Map:
		var tag byte
		switch v.(type) {
		case *ArrayMap:
			tag = constantArrayMap
		case *HashMap:
			tag = constantHashMap
		default:
			panic(RT.NewError(fmt.Sprintf("Cannot pack %T", obj)))
		}
		p = appendInt(append(p, tag), v.Count())
		for iter := v.Iter(); iter.HasNext(); {
			pair := iter.Next()
			p = packObject(pair.Key, p, env)
			p = packObject(pair.Value, p, env)
		}
	case *MapSet:
		p = append(p, constantSet)
		p = packObject(v.m, p, env)
	default:
		panic(RT.NewError(fmt.Sprintf("Cannot pack runtime constant %T", obj)))
	}
	p = obj.GetInfo().Pack(p, env)
	var meta Object
	if m, ok := obj.(Meta); ok {
		meta = m.GetMeta()
	}
	// A typed nil Map must be represented by a Go nil Object.
	if meta != nil && reflect.ValueOf(meta).Kind() == reflect.Ptr && reflect.ValueOf(meta).IsNil() {
		meta = nil
	}
	return PackObjectOrNull(meta, p, env)
}

func unpackObject(p []byte, h *PackHeader) (Object, []byte) {
	tag := p[0]
	p = p[1:]
	if tag == constantRef {
		id, rest := extractInt(p)
		if id < 0 || id >= len(h.objects) || h.objects[id] == nil {
			panic(RT.NewError("Invalid packed constant reference"))
		}
		return h.objects[id], rest
	}
	if tag == constantDef {
		id := len(h.objects)
		h.objects = append(h.objects, nil)
		obj, rest := unpackObject(p, h)
		h.objects[id] = obj
		return obj, rest
	}
	var obj Object
	switch tag {
	case constantNil:
		obj = NIL
	case constantBool:
		var b bool
		b, p = extractBool(p)
		obj = Boolean{B: b}
	case constantInt:
		var n int
		n, p = extractInt(p)
		obj = Int{I: n}
	case constantDouble:
		var n int
		n, p = extractInt(p)
		obj = Double{D: math.Float64frombits(uint64(n))}
	case constantChar:
		var n int
		n, p = extractInt(p)
		obj = Char{Ch: rune(n)}
	case constantString:
		var b []byte
		b, p = unpackBytes(p)
		obj = String{S: string(b)}
	case constantKeyword:
		var b []byte
		b, p = unpackBytes(p)
		obj = MakeKeyword(string(b))
	case constantSymbol:
		return unpackSymbol(p, h)
	case constantVar:
		return unpackVar(p, h)
	case constantType:
		return unpackType(p, h)
	case constantNamespace:
		s, rest := unpackSymbol(p, h)
		return h.GlobalEnv.EnsureSymbolIsNamespace(s), rest
	case constantBigInt:
		b, rest := unpackBytes(p)
		n := new(big.Int)
		PanicOnErr(n.GobDecode(b))
		obj = &BigInt{b: n}
		p = rest
	case constantBigFloat:
		b, rest := unpackBytes(p)
		n := new(big.Float)
		PanicOnErr(n.GobDecode(b))
		obj = &BigFloat{b: n}
		p = rest
	case constantRatio:
		b, rest := unpackBytes(p)
		n := new(big.Rat)
		PanicOnErr(n.GobDecode(b))
		obj = &Ratio{r: n}
		p = rest
	case constantRegex:
		b, rest := unpackBytes(p)
		r, err := regexp.Compile(string(b))
		PanicOnErr(err)
		obj = &Regex{R: r}
		p = rest
	case constantList:
		empty, rest := extractBool(p)
		p = rest
		if empty {
			obj = NewListFrom()
		} else {
			first, rest := unpackObject(p, h)
			tail, rest := unpackObject(rest, h)
			p = rest
			obj = NewList(first, tail.(*List))
		}
	case constantVectorSeq, constantVectorRSeq:
		i, rest := extractInt(p)
		v, rest := unpackObject(rest, h)
		p = rest
		if tag == constantVectorSeq {
			obj = &VectorSeq{vector: v.(CountedIndexed), index: i}
		} else {
			obj = &VectorRSeq{vector: v.(CountedIndexed), index: i}
		}
	case constantArraySeq:
		i, rest := extractInt(p)
		step, rest := extractInt(rest)
		n, rest := extractCount(rest)
		p = rest
		arr := make([]Object, n)
		for j := range arr {
			arr[j], p = unpackObject(p, h)
		}
		obj = &ArraySeq{arr: arr, index: i, step: step}
	case constantConsSeq:
		first, rest := unpackObject(p, h)
		tail, rest := unpackObject(rest, h)
		p = rest
		obj = &ConsSeq{first: first, rest: tail.(Seq)}
	case constantArrayMapSeq:
		i, rest := extractInt(p)
		m, rest := unpackObject(rest, h)
		p = rest
		obj = &ArrayMapSeq{m: m.(*ArrayMap), index: i}
	case constantArrayVector, constantVector:
		n, rest := extractCount(p)
		p = rest
		var v Vec = EmptyArrayVector()
		if tag == constantVector {
			v = EmptyVector()
		}
		for i := 0; i < n; i++ {
			var el Object
			el, p = unpackObject(p, h)
			v = v.Conj(el).(Vec)
		}
		obj = v
	case constantArrayMap, constantHashMap:
		n, rest := extractCount(p)
		p = rest
		var m Map = EmptyArrayMap()
		if tag == constantHashMap {
			m = EmptyHashMap
		}
		for i := 0; i < n; i++ {
			var k, v Object
			k, p = unpackObject(p, h)
			v, p = unpackObject(p, h)
			if arr, ok := m.(*ArrayMap); ok {
				arr.Add(k, v)
			} else {
				m = m.Assoc(k, v).(Map)
			}
		}
		obj = m
	case constantSet:
		m, rest := unpackObject(p, h)
		p = rest
		obj = &MapSet{m: m.(Map)}
	default:
		panic(RT.NewError(fmt.Sprintf("Unknown packed constant tag %d", tag)))
	}
	info, p := unpackObjectInfo(p, h)
	meta, p := UnpackObjectOrNull(p, h)
	if info != nil {
		obj = obj.WithInfo(info)
	}
	if meta != nil {
		obj = obj.(Meta).WithMeta(meta.(Map))
	}
	return obj, p
}
