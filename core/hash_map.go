package core

type (
	Box struct {
		val Object
	}
	Node interface {
		assoc(shift uint, hash uint32, key Object, val Object, addedLeaf *Box) Node
		without(shift uint, hash uint32, key Object) Node
		find(shift uint, hash uint32, key Object) *Pair
		nodeSeq() Seq
	}
	HashMap struct {
		InfoHolder
		MetaHolder
		count int
		root  Node
	}
	BitmapIndexedNode struct {
		bitmap int
		array  []interface{}
	}
	HashCollisionNode struct {
		hash  uint32
		count int
		array []interface{}
	}
	ArrayNode struct {
		count int
		array []Node
	}
	NodeSeq struct {
		InfoHolder
		MetaHolder
		array []interface{}
		i     int
		s     Seq
	}
	ArrayNodeSeq struct {
		InfoHolder
		MetaHolder
		nodes []Node
		i     int
		s     Seq
	}
)

var (
	EmptyHashMap     = &HashMap{}
	emptyIndexedNode = &BitmapIndexedNode{}
	notFound         = EmptyArrayMap()
)

func newArrayNodeSeq(nodes []Node, i int, s Seq) Seq {
	if s != nil {
		return &ArrayNodeSeq{
			nodes: nodes,
			i:     i,
			s:     s,
		}
	}
	for j := i; j < len(nodes); j++ {
		if nodes[j] != nil {
			ns := nodes[j].nodeSeq()
			if ns != nil {
				return &ArrayNodeSeq{
					nodes: nodes,
					i:     j + 1,
					s:     ns,
				}
			}
		}
	}
	return nil
}

func (s *ArrayNodeSeq) WithMeta(meta *ArrayMap) Object {
	res := *s
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (s *ArrayNodeSeq) Seq() Seq {
	return s
}

func (s *ArrayNodeSeq) Equals(other interface{}) bool {
	return IsSeqEqual(s, other)
}

func (s *ArrayNodeSeq) ToString(escape bool) string {
	return SeqToString(s, escape)
}

func (s *ArrayNodeSeq) GetType() *Type {
	return TYPES["ArrayNodeSeq"]
}

func (s *ArrayNodeSeq) Hash() uint32 {
	return hashOrdered(s)
}

func (s *ArrayNodeSeq) First() Object {
	return s.First()
}

func (s *ArrayNodeSeq) Rest() Seq {
	res := newArrayNodeSeq(s.nodes, s.i, s.Rest())
	if res == nil {
		return EmptyList
	}
	return res
}

func (s *ArrayNodeSeq) IsEmpty() bool {
	if s.s != nil {
		return s.IsEmpty()
	}
	return false
}

func (s *ArrayNodeSeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: s}
}

func (s *ArrayNodeSeq) sequential() {}

func newNodeSeq(array []interface{}, i int, s Seq) Seq {
	if s != nil {
		return &NodeSeq{
			array: array,
			i:     i,
			s:     s,
		}
	}
	for j := i; j < len(array); j += 2 {
		if array[j] != nil {
			return &NodeSeq{
				array: array,
				i:     j,
			}
		}
		switch node := array[j+1].(type) {
		case Node:
			nodeSeq := node.nodeSeq()
			if nodeSeq != nil {
				return &NodeSeq{
					array: array,
					i:     j + 2,
					s:     nodeSeq,
				}
			}
		}
	}
	return nil
}

func (s *NodeSeq) WithMeta(meta *ArrayMap) Object {
	res := *s
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (s *NodeSeq) Seq() Seq {
	return s
}

func (s *NodeSeq) Equals(other interface{}) bool {
	return IsSeqEqual(s, other)
}

func (s *NodeSeq) ToString(escape bool) string {
	return SeqToString(s, escape)
}

func (s *NodeSeq) GetType() *Type {
	return TYPES["NodeSeq"]
}

func (s *NodeSeq) Hash() uint32 {
	return hashOrdered(s)
}

func (s *NodeSeq) First() Object {
	if s.s != nil {
		return s.First()
	}
	return NewVectorFrom(s.array[s.i].(Object), s.array[s.i+1].(Object))
}

func (s *NodeSeq) Rest() Seq {
	var res Seq
	if s.s != nil {
		res = newNodeSeq(s.array, s.i, s.Rest())
	} else {
		res = newNodeSeq(s.array, s.i+2, nil)
	}
	if res == nil {
		return EmptyList
	}
	return res
}

func (s *NodeSeq) IsEmpty() bool {
	if s.s != nil {
		return s.IsEmpty()
	}
	return false
}

func (s *NodeSeq) Cons(obj Object) Seq {
	return &ConsSeq{first: obj, rest: s}
}

func (s *NodeSeq) sequential() {}

func (n *HashCollisionNode) findIndex(key Object) int {
	for i := 0; i < 2*n.count; i += 2 {
		if key.Equals(n.array[i]) {
			return i
		}
	}
	return -1
}

func (n *ArrayNode) assoc(shift uint, hash uint32, key Object, val Object, addedLeaf *Box) Node {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		return &ArrayNode{
			count: n.count + 1,
			array: cloneAndSetNode(n.array, int(idx), emptyIndexedNode.assoc(shift+5, hash, key, val, addedLeaf)),
		}
	}
	nn := node.assoc(shift+5, hash, key, val, addedLeaf)
	if nn == node {
		return n
	}
	return &ArrayNode{
		count: n.count,
		array: cloneAndSetNode(n.array, int(idx), nn),
	}
}

func (n *ArrayNode) without(shift uint, hash uint32, key Object) Node {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		return n
	}
	nn := node.without(shift+5, hash, key)
	if nn == node {
		return n
	}
	if nn == nil {
		if n.count <= 8 {
			return n.pack(uint(idx))
		}
		return &ArrayNode{
			count: n.count - 1,
			array: cloneAndSetNode(n.array, int(idx), nn),
		}
	} else {
		return &ArrayNode{
			count: n.count,
			array: cloneAndSetNode(n.array, int(idx), nn),
		}
	}
}

func (n *ArrayNode) find(shift uint, hash uint32, key Object) *Pair {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		return nil
	}
	return node.find(shift+5, hash, key)
}

func (n *ArrayNode) nodeSeq() Seq {
	return newArrayNodeSeq(n.array, 0, nil)
}

func (n *ArrayNode) pack(idx uint) Node {
	newArray := make([]interface{}, 2*(n.count-1))
	j := 1
	bitmap := 0
	var i uint
	for i = 0; i < idx; i++ {
		if n.array[i] != nil {
			newArray[j] = n.array[i]
			bitmap |= 1 << i
			j += 2
		}
	}
	for i = idx + 1; i < uint(len(n.array)); i++ {
		if n.array[i] != nil {
			newArray[j] = n.array[i]
			bitmap |= 1 << i
			j += 2
		}
	}
	return &BitmapIndexedNode{
		bitmap: bitmap,
		array:  newArray,
	}
}

func (n *HashCollisionNode) assoc(shift uint, hash uint32, key Object, val Object, addedLeaf *Box) Node {
	if hash == n.hash {
		idx := n.findIndex(key)
		if idx != -1 {
			if n.array[idx+1] == val {
				return n
			}
			return &HashCollisionNode{
				hash:  hash,
				count: n.count,
				array: cloneAndSet(n.array, idx+1, val),
			}
		}
		newArray := make([]interface{}, 2*(n.count+1))
		for i := 0; i < 2*n.count; i++ {
			newArray[i] = n.array[i]
		}
		newArray[2*n.count] = key
		newArray[2*n.count+1] = val
		// addedLeaf.val = addedLeaf
		return &HashCollisionNode{
			hash:  hash,
			count: n.count + 1,
			array: newArray,
		}
	}
	return (&BitmapIndexedNode{
		bitmap: bitpos(n.hash, shift),
		array:  []interface{}{nil, n},
	}).assoc(shift, hash, key, val, addedLeaf)
}

func (n *HashCollisionNode) without(shift uint, hash uint32, key Object) Node {
	idx := n.findIndex(key)
	if idx == -1 {
		return n
	}
	if n.count == 1 {
		return nil
	}
	return &HashCollisionNode{
		hash:  hash,
		count: n.count - 1,
		array: removePair(n.array, idx/2),
	}
}

func (n *HashCollisionNode) find(shift uint, hash uint32, key Object) *Pair {
	idx := n.findIndex(key)
	if idx == -1 {
		return nil
	}
	return &Pair{
		key:   n.array[idx].(Object),
		value: n.array[idx+1].(Object),
	}
}

func (n *HashCollisionNode) nodeSeq() Seq {
	return newNodeSeq(n.array, 0, nil)
}

func bitCount(n int) int {
	var count int
	for n != 0 {
		count++
		n &= n - 1
	}
	return count
}

func mask(hash uint32, shift uint) uint32 {
	return (hash >> shift) & 0x01f
}

func bitpos(hash uint32, shift uint) int {
	return 1 << mask(hash, shift)
}

func cloneAndSet(array []interface{}, i int, a interface{}) []interface{} {
	res := clone(array)
	res[i] = a
	return res
}

func cloneAndSet2(array []interface{}, i int, a interface{}, j int, b interface{}) []interface{} {
	res := clone(array)
	res[i] = a
	res[j] = b
	return res
}

func cloneAndSetNode(array []Node, i int, a Node) []Node {
	res := make([]Node, len(array), cap(array))
	copy(res, array)
	res[i] = a
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

func (b *BitmapIndexedNode) index(bit int) int {
	return bitCount(b.bitmap & (bit - 1))
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
			newArray := make([]interface{}, 2*(n+1))
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

func removePair(array []interface{}, n int) []interface{} {
	newArray := make([]interface{}, len(array)-2)
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

func (b *BitmapIndexedNode) find(shift uint, hash uint32, key Object) *Pair {
	bit := bitpos(hash, shift)
	if (b.bitmap & bit) == 0 {
		return nil
	}
	idx := b.index(bit)
	keyOrNull := b.array[2*idx]
	valOrNode := b.array[2*idx+1]
	if keyOrNull == nil {
		return valOrNode.(Node).find(shift+5, hash, key)
	}
	if key.Equals(keyOrNull) {
		return &Pair{
			key:   keyOrNull.(Object),
			value: valOrNode.(Object),
		}
	}
	return nil
}

func (b *BitmapIndexedNode) nodeSeq() Seq {
	return &NodeSeq{array: b.array}
}

func (m *HashMap) containsKey(key Object) bool {
	if m.root != nil {
		return m.root.find(0, key.Hash(), key) != nil
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
