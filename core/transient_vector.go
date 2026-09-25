package core

import "fmt"

// TransientVector uses an array for small vectors and an editable trie for large ones.
// Only nodes owned by this transient may be changed in place; shared persistent nodes
// are copied on the first write. The ownership table is discarded at persistent!.
type TransientVector struct {
	InfoHolder
	live  bool
	arr   []Object
	root  []interface{}
	tail  []interface{}
	count int
	shift uint
	owned map[*interface{}]bool
}

func (t *TransientVector) check() {
	if !t.live {
		panic(RT.NewError("Transient used after persistent! call"))
	}
}

func (t *TransientVector) GetType() *Type                { return TYPE.TransientVector }
func (t *TransientVector) ToString(bool) string          { return "#<transient vector>" }
func (t *TransientVector) Equals(other interface{}) bool { return t == other }
func (t *TransientVector) Hash() uint32                  { panic(RT.NewError("Cannot hash a transient vector")) }

func (v *Vector) AsTransient() TransientCollection {
	tail := make([]interface{}, len(v.tail), 32)
	copy(tail, v.tail)
	return &TransientVector{live: true, root: v.root, tail: tail, count: v.count, shift: v.shift, owned: make(map[*interface{}]bool)}
}

func (v *ArrayVector) AsTransient() TransientCollection {
	arr := make([]Object, len(v.arr))
	copy(arr, v.arr)
	return &TransientVector{live: true, arr: arr}
}

func (t *TransientVector) Count() int {
	t.check()
	if t.arr != nil {
		return len(t.arr)
	}
	return t.count
}

func (t *TransientVector) tailoff() int {
	if t.count < 32 {
		return 0
	}
	return ((t.count - 1) >> 5) << 5
}

func (t *TransientVector) arrayFor(i int) []interface{} {
	if i >= t.tailoff() {
		return t.tail
	}
	node := t.root
	for level := t.shift; level > 0; level -= 5 {
		node = node[(i>>level)&31].([]interface{})
	}
	return node
}

func (t *TransientVector) Nth(i int) Object {
	t.check()
	n := t.Count()
	if i < 0 || i >= n {
		panic(RT.NewError(fmt.Sprintf("Index %d is out of bounds [0..%d]", i, n-1)))
	}
	if t.arr != nil {
		return t.arr[i]
	}
	return t.arrayFor(i)[i&31].(Object)
}

func (t *TransientVector) TryNth(i int, d Object) Object {
	t.check()
	if i < 0 || i >= t.Count() {
		return d
	}
	return t.Nth(i)
}

func (t *TransientVector) At(i int) Object { return t.Nth(i) }
func (t *TransientVector) Get(key Object) (bool, Object) {
	t.check()
	return CountedIndexedGet(t, key)
}
func (t *TransientVector) Call(args []Object) Object {
	CheckArity(args, 1, 1)
	return t.Nth(assertInteger(args[0]))
}

func (t *TransientVector) own(node []interface{}) []interface{} {
	if t.owned[&node[0]] {
		return node
	}
	res := clone(node)
	t.owned[&res[0]] = true
	return res
}

func (t *TransientVector) path(level uint, node []interface{}) []interface{} {
	if level == 0 {
		return node
	}
	res := make([]interface{}, 32)
	t.owned[&res[0]] = true
	res[0] = t.path(level-5, node)
	return res
}

func (t *TransientVector) pushTail(level uint, parent, tail []interface{}) []interface{} {
	parent = t.own(parent)
	idx := ((t.count - 1) >> level) & 31
	if level == 5 {
		parent[idx] = tail
	} else if parent[idx] != nil {
		parent[idx] = t.pushTail(level-5, parent[idx].([]interface{}), tail)
	} else {
		parent[idx] = t.path(level-5, tail)
	}
	return parent
}

func (t *TransientVector) ConjBang(obj Object) TransientCollection {
	t.check()
	if t.arr != nil {
		if len(t.arr) < VECTOR_THRESHOLD {
			t.arr = append(t.arr, obj)
			return t
		}
		// Conversion is bounded by the small-vector threshold.
		v := NewVectorFrom(t.arr...).AsTransient().(*TransientVector)
		t.arr, t.root, t.tail, t.count, t.shift, t.owned = nil, v.root, v.tail, v.count, v.shift, v.owned
	}
	if t.count-t.tailoff() < 32 {
		t.tail = append(t.tail, obj)
		t.count++
		return t
	}
	tailNode := make([]interface{}, 32)
	copy(tailNode, t.tail)
	t.owned[&tailNode[0]] = true
	if (t.count >> 5) > (1 << t.shift) {
		root := make([]interface{}, 32)
		t.owned[&root[0]] = true
		root[0] = t.root
		root[1] = t.path(t.shift, tailNode)
		t.root = root
		t.shift += 5
	} else {
		t.root = t.pushTail(t.shift, t.root, tailNode)
	}
	t.tail = make([]interface{}, 1, 32)
	t.tail[0] = obj
	t.count++
	return t
}

func (t *TransientVector) assocNode(level uint, node []interface{}, i int, val Object) []interface{} {
	node = t.own(node)
	if level == 0 {
		node[i&31] = val
	} else {
		idx := (i >> level) & 31
		node[idx] = t.assocNode(level-5, node[idx].([]interface{}), i, val)
	}
	return node
}

func (t *TransientVector) AssocBang(key, val Object) TransientAssociative {
	t.check()
	i := assertInteger(key)
	n := t.Count()
	if i < 0 || i > n {
		panic(RT.NewError(fmt.Sprintf("Index %d is out of bounds [0..%d]", i, n)))
	}
	if i == n {
		t.ConjBang(val)
		return t
	}
	if t.arr != nil {
		t.arr[i] = val
		return t
	}
	if i >= t.tailoff() {
		t.tail[i&31] = val
	} else {
		t.root = t.assocNode(t.shift, t.root, i, val)
	}
	return t
}

func (t *TransientVector) popTail(level uint, node []interface{}) []interface{} {
	idx := ((t.count - 2) >> level) & 31
	if level > 5 {
		child := t.popTail(level-5, node[idx].([]interface{}))
		if child == nil && idx == 0 {
			return nil
		}
		node = t.own(node)
		if child == nil {
			node[idx] = nil
		} else {
			node[idx] = child
		}
		return node
	}
	if idx == 0 {
		return nil
	}
	node = t.own(node)
	node[idx] = nil
	return node
}

func (t *TransientVector) PopBang() *TransientVector {
	t.check()
	if t.Count() == 0 {
		panic(RT.NewError("Can't pop empty vector"))
	}
	if t.arr != nil {
		t.arr[len(t.arr)-1] = nil
		t.arr = t.arr[:len(t.arr)-1]
		return t
	}
	if t.count == 1 {
		t.count = 0
		t.tail = t.tail[:0]
		return t
	}
	if t.count-t.tailoff() > 1 {
		t.tail[len(t.tail)-1] = nil
		t.tail = t.tail[:len(t.tail)-1]
		t.count--
		return t
	}
	newTail := make([]interface{}, 32)
	copy(newTail, t.arrayFor(t.count-2))
	root := t.popTail(t.shift, t.root)
	if root == nil {
		root = empty_node
	}
	if t.shift > 5 && root[1] == nil {
		root = root[0].([]interface{})
		t.shift -= 5
	}
	t.root, t.tail = root, newTail[:((t.count-2)&31)+1]
	t.count--
	return t
}

func (t *TransientVector) Persistent() Object {
	t.check()
	t.live = false
	if t.arr != nil {
		return &ArrayVector{arr: t.arr}
	}
	return &Vector{root: t.root, tail: t.tail, count: t.count, shift: t.shift}
}
