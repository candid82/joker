package net

import "fmt"
import "net"

import (
	. "github.com/candid82/joker/core"
)

func lookupMX(s string) Object {
	mxen, e := net.LookupMX(s)
	mxinfo := EmptyVector
	for _, mx := range mxen {
		mxmap := EmptyArrayMap() // { :Host hostname, :Pref preference }
		mxmap.Add(MakeKeyword("Host"), String{S: mx.Host})
		mxmap.Add(MakeKeyword("Pref"), Int{I: int(mx.Pref)})
		mxinfo = mxinfo.Conjoin(mxmap)
	}
	res := EmptyVector
	res = res.Conjoin(mxinfo)
	var err Object
	if e == nil {
		err = NIL
	} else {
		err = String{S: fmt.Sprintf("%s", e)}
	}
	res = res.Conjoin(err)
	return res // [ mxinfo err ]
}
