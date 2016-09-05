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
		count int
		root  Node
	}
	BitmapIndexedNode struct {
		bitmap int
		array  []Object
	}
)

var (
	EmptyHashMap     = &HashMap{}
	emptyIndexedNode = &BitmapIndexedNode{}
	notFound         = EmptyArrayMap()
)

func bitCount(n int) int {
	var count int
	for n != 0 {
		count++
		n &= n - 1
	}
	return count
}

func (b *BitmapIndexedNode) index(bit int) int {
	return bitCount(b.bitmap & (bit - 1))
}

func (m *HashMap) containsKey(key Object) bool {
	if m.root != nil {
		return m.root.tryFind(0, int(key.Hash()), key, notFound) != notFound
	} else {
		return false
	}
}

// func (m *HashMap) Assoc(key, val Object) Associative {
// 	addedLeaf := &Box{}
// 	var newroot, t Node
// 	if m.root == nil {
// 		t = EmptyBitmapIndexedNode
// 	} else {
// 		t = root
// 	}
// 	newroot = t.assoc(0, key.Hash(), key, val, addedLeaf)
// 	if newroot == root {
// 		return m
// 	}
// 	newcount := m.count
// 	if addedLeaf.val != nil {
// 		newcount = m.count + 1
// 	}
// 	return &HashMap{
// 		count: newcount,
// 		root:  newroot,
// 		meta:  m.meta,
// 	}

// }
