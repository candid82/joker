package core

import "testing"

func TestVMProcDispatchAllocations(t *testing.T) {
	for _, pkg := range []string{"", "test"} {
		t.Run(pkg, func(t *testing.T) {
			var retained []Object
			var callee Object = Proc{Package: pkg, Fn: func(args []Object) Object {
				if pkg != "" {
					retained = args
				}
				return args[0]
			}}
			vm := NewVM()
			var arg Object = MakeKeyword("argument")
			allocs := testing.AllocsPerRun(20, func() {
				vm.Push(callee)
				vm.Push(arg)
				if vm.callValue(callee, 1) || vm.Pop() != arg {
					t.Fatal("incorrect native dispatch result")
				}
			})
			budget := float64(0)
			if pkg != "" {
				budget = 1 // The independently owned argument slice, not a boxed Proc.
				vm.Push(NIL)
				vm.Push(NIL)
				vm.Reset()
				if len(retained) != 1 || retained[0] != arg {
					t.Fatal("std procedure arguments overwritten by stack reuse")
				}
			}
			if allocs > budget {
				t.Fatalf("%.0f allocations per native call; budget is %.0f", allocs, budget)
			}
		})
	}
}

func BenchmarkVMStdProcDispatch(b *testing.B) {
	var callee Object = Proc{Package: "test", Fn: func(args []Object) Object { return args[0] }}
	vm := NewVM()
	var arg Object = NIL
	vm.Push(arg) // Warm stack storage before timing.
	vm.Reset()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.Push(callee)
		vm.Push(arg)
		vm.callValue(callee, 1)
		vm.Pop()
	}
}
