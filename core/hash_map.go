package core

type (
	Box struct {
		val Object
	}
	Node interface {
		Object
		assoc(shift uint, hash uint32, key Object, val Object, addedLeaf *Box) Node
		without(shift uint, hash uint32, key Object) Node
		find(shift uint, hash uint32, key Object) Pair
		tryFind(shift uint, hash uint32, key Object, notFound Object) Object
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

func mask(hash uint32, shift uint) uint32 {
	return (hash >> shift) & 0x01f
}

func bitpos(hash uint32, shift uint) int {
	return 1 << mask(hash, shift)
}

func cloneObjects(s []Object) []Object {
	result := make([]Object, len(s), cap(s))
	copy(result, s)
	return result
}

func cloneAndSet(array []Object, i int, a Object) []Object {
	res := cloneObjects(array)
	res[i] = a
	return res
}

func cloneAndSet2(array []Object, i int, a Object, j int, b Object) []Object {
	res := cloneObjects(array)
	res[i] = a
	res[j] = b
	return res
}

func createNode(shift uint, key1 Object, val1 Object, key2hash uint32, key2 Object, val2 Object) Node {
	key1hash := key1.Hash()
	if key1hash == key2hash {
		return &HashCollisionNode{}
	}
	addedLeaf := &Box{}
	return emptyIndexedNode.assoc(shift, key1hash, key1, val1, addedLeaf).assoc(shift, key2hash, key2, val2, addedLeaf)
}

func (b *BitmapIndexedNode) assoc(shift uint, hash uint32, key Object, val Object, addedLeaf *Box) Node {
	bit := bitpos(hash, shift)
	idx := b.index(bit)
	if b.bitmap&bit != 0 {
		keyOrNull := b.array[2*idx]
		valOrNode := b.array[2*idx+1]
		if keyOrNull == nil {
			n := valOrNode.(Node).assoc(shift+5, hash, key, val, addedLeaf)
			if n == valOrNode {
				return b
			}
			return &BitmapIndexedNode{
				bitmap: b.bitmap,
				array:  cloneAndSet(b.array, 2*idx+1, n),
			}
		}
		if key.Equals(keyOrNull) {
			if val == valOrNode {
				return b
			}
			return &BitmapIndexedNode{
				bitmap: b.bitmap,
				array:  cloneAndSet(b.array, 2*idx+1, val),
			}
		}
		// addedLeaf.val = addedLeaf
		return &BitmapIndexedNode{
			bitmap: b.bitmap,
			array:  cloneAndSet2(b.array, 2*idx, nil, 2*idx+1, createNode(shift+5, keyOrNull.(Object), valOrNode.(Object), hash, key, val)),
		}
	} else {
		n := bitCount(b.bitmap)
		if n >= 16 {
			nodes := make([]Node, 32)
			jdx := mask(hash, shift)
			nodes[jdx] = emptyIndexedNode.assoc(shift+5, hash, key, val, addedLeaf)
			j := 0
			var i uint
			for i = 0; i < 32; i++ {
				if (b.bitmap>>i)&1 != 0 {
					if b.array[j] == nil {
						nodes[i] = b.array[j+1].(Node)
					} else {
						nodes[i] = emptyIndexedNode.assoc(shift+5, b.array[j].(Object).Hash(), b.array[j].(Object), b.array[j+1].(Object), addedLeaf)
					}
					j += 2
				}
			}
			return &ArrayNode{}
		} else {
			newArray := make([]Object, 2*(n+1))
			for i := 0; i < 2*idx; i++ {
				newArray[i] = b.array[i]
			}
			newArray[2*idx] = key
			// addedLeaf.val = addedLeaf
			newArray[2*idx+1] = val
			for i := 2 * idx; i < 2*n; i++ {
				newArray[i+2] = b.array[i].(Object)
			}
			return &BitmapIndexedNode{
				bitmap: b.bitmap | bit,
				array:  newArray,
			}
		}
	}
}

func removePair(array []Object, n int) []Object {
	newArray := make([]Object, len(array)-2)
	for i := 0; i < 2*n; i++ {
		newArray[i] = array[i]
	}
	for i := 2 * (n + 1); i < len(array); i++ {
		newArray[i-2] = array[i]
	}
	return newArray
}

func (b *BitmapIndexedNode) without(shift uint, hash uint32, key Object) Node {
	bit := bitpos(hash, shift)
	if (b.bitmap & bit) == 0 {
		return b
	}
	idx := b.index(bit)
	keyOrNull := b.array[2*idx]
	valOrNode := b.array[2*idx+1]
	if keyOrNull == nil {
		n := valOrNode.(Node).without(shift+5, hash, key)
		if n == valOrNode {
			return b
		}
		if n != nil {
			return &BitmapIndexedNode{
				bitmap: b.bitmap,
				array:  cloneAndSet(b.array, 2*idx+1, n),
			}
		}
		if b.bitmap == bit {
			return nil
		}
		return &BitmapIndexedNode{
			bitmap: b.bitmap ^ bit,
			array:  removePair(b.array, idx),
		}
	}
	if key.Equals(keyOrNull) {
		return &BitmapIndexedNode{
			bitmap: b.bitmap ^ bit,
			array:  removePair(b.array, idx),
		}
	}
	return b
}

func (m *HashMap) containsKey(key Object) bool {
	if m.root != nil {
		return m.root.tryFind(0, key.Hash(), key, notFound) != notFound
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
