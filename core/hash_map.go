package core

type (
	Box struct {
		val Object
	}
	Node interface {
		assoc(shift int, hash int, key Object, val Object, addedLeaf Box) Node
		without(shift int, hash int, key Object) Node
		find(shift int, hash int, key Object) Pair
		tryFind(shift int, hash int, key Object, notFound Object) Object
		nodeSeq() Seq
		kvreduce(f Callable, init Object) Object
		fold(combinef Callable, reducef Callable, fjtask Callable, fjfork Callable, fjjoin Callable) Object
	}
	HashMap struct {
		InfoHolder
		MetaHolder
		count     int
		root      Node
		hasNull   bool
		nullValue Object
	}
)

var EmptyHashMap = &HashMap{}
