package net

import "fmt"
import "net"

import (
	. "github.com/candid82/joker/core"
)

func lookupMX(s string) Map {
	mxen, e := net.LookupMX(s)
	mxinfo := EmptyVector
	for _, mx := range mxen {
		mxmap := EmptyArrayMap() // { :Host hostname, :Pref preference }
		mxmap.Add(MakeKeyword("Host"), String{S: mx.Host})
		mxmap.Add(MakeKeyword("Pref"), Int{I: int(mx.Pref)})
		mxinfo = mxinfo.Conjoin(mxmap)
	}
	ret := EmptyArrayMap()
	ret.Add(MakeKeyword("res"), mxinfo)
	if e == nil {
		ret.Add(MakeKeyword("err"), NIL)
	} else {
		ret.Add(MakeKeyword("err"), String{S: fmt.Sprintf("%s", e)})
	}
	return ret // { :res mxinfo, :err err }
}
