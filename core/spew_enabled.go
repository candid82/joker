// +build go_spew

package core

import (
	"fmt"
	"github.com/jcburley/go-spew/spew"
)

var procGoSpew Proc = func(args []Object) (res Object) {
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
