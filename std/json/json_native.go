package json

import (
	"encoding/json"
	"fmt"
	. "github.com/candid82/joker/core"
	"io"
	"strings"
)

func fromObject(obj Object) interface{} {
	switch obj := obj.(type) {
	case Keyword:
		return obj.ToString(false)[1:]
	case Boolean:
		return obj.B
	case Number:
		return obj.Double().D
	case Nil:
		return nil
	case String:
		return obj.ToString(false)
	case Map:
		res := make(map[string]interface{})
		for iter := obj.Iter(); iter.HasNext(); {
			p := iter.Next()
			var k string
			switch p.Key.(type) {
			case Keyword:
				k = p.Key.ToString(false)[1:]
			default:
				k = p.Key.ToString(false)
			}
			res[k] = fromObject(p.Value)
		}
		return res
	case Seqable:
		s := obj.Seq()
		var res []interface{}
		for !s.IsEmpty() {
			res = append(res, fromObject(s.First()))
			s = s.Rest()
		}
		return res
	default:
		return obj.ToString(false)
	}
}

func toObject(v interface{}, keywordize bool) Object {
	switch v := v.(type) {
	case string:
		return MakeString(v)
	case float64:
		if v == float64(int(v)) {
			return Int{I: int(v)}
		}
		return Double{D: v}
	case bool:
		return Boolean{B: v}
	case nil:
		return NIL
	case []interface{}:
		res := EmptyVector()
		for _, v := range v {
			res = res.Conjoin(toObject(v, keywordize))
		}
		return res
	case map[string]interface{}:
		res := EmptyArrayMap()
		for k, v := range v {
			var key Object
			if keywordize {
				key = MakeKeyword(k)
			} else {
				key = MakeString(k)
			}
			res.Add(key, toObject(v, keywordize))
		}
		return res
	default:
		panic(RT.NewError(fmt.Sprintf("Unknown json value: %v", v)))
	}
}

func readString(s string, opts Map) Object {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic(RT.NewError("Invalid json: " + err.Error()))
	}
	var keywordize bool
	if opts != nil {
		if ok, v := opts.Get(MakeKeyword("keywords?")); ok {
			keywordize = ToBool(v)
		}
	}
	return toObject(v, keywordize)
}

func jsonSeqOpts(src Object, opts Map) Object {
	var dec *json.Decoder
	var keywordize bool
	var jsonLazySeq func() *LazySeq
	switch src := src.(type) {
	case String:
		dec = json.NewDecoder(strings.NewReader(src.S))
	case io.Reader:
		dec = json.NewDecoder(src)
	default:
		panic(RT.NewError("src must be a string or io.Reader"))
	}
	if opts != nil {
		if ok, v := opts.Get(MakeKeyword("keywords?")); ok {
			keywordize = ToBool(v)
		}
	}
	jsonLazySeq = func() *LazySeq {
		var c = func(args []Object) Object {
			var o interface{}
			err := dec.Decode(&o)
			if err == io.EOF {
				return EmptyList
			}
			PanicOnErr(err)
			obj := toObject(o, keywordize)
			return NewConsSeq(obj, jsonLazySeq())
		}
		return NewLazySeq(Proc{Fn: c})
	}
	return jsonLazySeq()
}

func writeString(obj Object) String {
	res, err := json.Marshal(fromObject(obj))
	if err != nil {
		panic(RT.NewError("Cannot encode value to json: " + err.Error()))
	}
	return String{S: string(res)}
}
