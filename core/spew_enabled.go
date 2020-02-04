// +build go_spew

package core

import (
	"fmt"
	"github.com/jcburley/go-spew/spew"
)

var mySpewState spew.SpewState

func Spew() {
	cs := &spew.ConfigState{
		Indent:            "    ",
		MaxDepth:          30,
		SortKeys:          true,
		SpewKeys:          true,
		NoDuplicates:      true,
		UseOrdinals:       true,
		DisableMethods:    true,
		PreserveSpewState: true,
		SpewState:         mySpewState,
	}

	cs.Fprintln(Stderr, "STR:")
	cs.Fdump(Stderr, STR)
	cs.Fprintln(Stderr, "STRINGS:")
	cs.Fdump(Stderr, STRINGS)
	cs.Fprintln(Stderr, "\nSYMBOLS:")
	cs.Fdump(Stderr, SYMBOLS)
	cs.Fprintln(Stderr, "\nSPECIAL_SYMBOLS:")
	cs.Fdump(Stderr, SPECIAL_SYMBOLS)
	cs.Fprintln(Stderr, "\nKEYWORDS:")
	cs.Fdump(Stderr, KEYWORDS)
	cs.Fprintln(Stderr, "\nTYPE:")
	cs.Fdump(Stderr, TYPE)
	cs.Fprintln(Stderr, "\nTYPES:")
	cs.Fdump(Stderr, TYPES)
	cs.Fprintln(Stderr, "\nGLOBAL_ENV:")
	cs.Fdump(Stderr, GLOBAL_ENV)
	mySpewState = cs.SpewState
}

func SpewThis(obj interface{}) {
	cs := &spew.ConfigState{
		Indent:            "    ",
		MaxDepth:          30,
		SortKeys:          true,
		SpewKeys:          true,
		NoDuplicates:      true,
		UseOrdinals:       true,
		DisableMethods:    true,
		PreserveSpewState: true,
		SpewState:         mySpewState,
	}

	cs.Fdump(Stderr, obj)
	mySpewState = cs.SpewState
}

func SpewObj(obj interface{}) string {
	cs := &spew.ConfigState{
		Indent:            "    ",
		MaxDepth:          30,
		SortKeys:          true,
		SpewKeys:          true,
		NoDuplicates:      true,
		UseOrdinals:       true,
		DisableMethods:    true,
		PreserveSpewState: true,
		SpewState:         mySpewState,
	}

	res := cs.Sdump(obj)
	mySpewState = cs.SpewState
	return res
}

var procGoSpew = func(args []Object) (res Object) {
	res = MakeBoolean(false)
	CheckArity(args, 1, 2)
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(Stderr, "Error: %v\n", r)
		}
	}()
	scs := spew.NewDefaultConfig()
	if len(args) > 1 {
		m := ExtractMap(args, 1)
		if yes, k := m.Get(MakeKeyword("Indent")); yes {
			scs.Indent = k.(Native).Native().(string)
		}
		if yes, k := m.Get(MakeKeyword("MaxDepth")); yes {
			scs.MaxDepth = k.(Native).Native().(int)
		}
		if yes, k := m.Get(MakeKeyword("DisableMethods")); yes {
			scs.DisableMethods = k.(Native).Native().(bool)
		}
		if yes, k := m.Get(MakeKeyword("DisablePointerMethods")); yes {
			scs.DisablePointerMethods = k.(Native).Native().(bool)
		}
		if yes, k := m.Get(MakeKeyword("DisablePointerAddresses")); yes {
			scs.DisablePointerAddresses = k.(Native).Native().(bool)
		}
		if yes, k := m.Get(MakeKeyword("DisableCapacities")); yes {
			scs.DisableCapacities = k.(Native).Native().(bool)
		}
		if yes, k := m.Get(MakeKeyword("ContinueOnMethod")); yes {
			scs.ContinueOnMethod = k.(Native).Native().(bool)
		}
		if yes, k := m.Get(MakeKeyword("SortKeys")); yes {
			scs.SortKeys = k.(Native).Native().(bool)
		}
		if yes, k := m.Get(MakeKeyword("SpewKeys")); yes {
			scs.SpewKeys = k.(Native).Native().(bool)
		}
		if yes, k := m.Get(MakeKeyword("NoDuplicates")); yes {
			scs.NoDuplicates = k.(Native).Native().(bool)
		}
		if yes, k := m.Get(MakeKeyword("UseOrdinals")); yes {
			scs.UseOrdinals = k.(Native).Native().(bool)
		}
	}
	scs.Fdump(Stderr, args[0])
	return MakeBoolean(true)
}
