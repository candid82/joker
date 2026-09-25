package core

// A transient edit token belongs to exactly one hash-map builder. Persistent
// nodes have no token, so they are copied before mutation by a transient.
type transientEdit struct{ marker int }

type TransientArrayMap struct {
	InfoHolder
	live bool
	arr  []Object
}

type TransientHashMap struct {
	InfoHolder
	live  bool
	edit  *transientEdit
	count int
	root  Node
}

func (m *ArrayMap) AsTransient() TransientCollection {
	arr := make([]Object, len(m.arr))
	copy(arr, m.arr)
	return &TransientArrayMap{live: true, arr: arr}
}

func (m *HashMap) AsTransient() TransientCollection {
	return &TransientHashMap{live: true, edit: &transientEdit{}, count: m.count, root: m.root}
}

func (m *TransientArrayMap) check() {
	if !m.live {
		panic(RT.NewError("Transient used after persistent! call"))
	}
}
func (m *TransientHashMap) check() {
	if !m.live {
		panic(RT.NewError("Transient used after persistent! call"))
	}
}

func (m *TransientArrayMap) GetType() *Type                { return TYPE.TransientArrayMap }
func (m *TransientHashMap) GetType() *Type                 { return TYPE.TransientHashMap }
func (m *TransientArrayMap) ToString(bool) string          { return "#<transient array map>" }
func (m *TransientHashMap) ToString(bool) string           { return "#<transient hash map>" }
func (m *TransientArrayMap) Equals(other interface{}) bool { return m == other }
func (m *TransientHashMap) Equals(other interface{}) bool  { return m == other }
func (m *TransientArrayMap) Hash() uint32                  { panic(RT.NewError("Cannot hash a transient map")) }
func (m *TransientHashMap) Hash() uint32                   { panic(RT.NewError("Cannot hash a transient map")) }

func (m *TransientArrayMap) indexOf(key Object) int {
	for i := 0; i < len(m.arr); i += 2 {
		if m.arr[i].Equals(key) {
			return i
		}
	}
	return -1
}
func (m *TransientArrayMap) Count() int { m.check(); return len(m.arr) / 2 }
func (m *TransientArrayMap) Get(key Object) (bool, Object) {
	m.check()
	if i := m.indexOf(key); i >= 0 {
		return true, m.arr[i+1]
	}
	return false, nil
}
func (m *TransientArrayMap) Call(args []Object) Object {
	CheckArity(args, 1, 2)
	if ok, val := m.Get(args[0]); ok {
		return val
	}
	if len(args) == 2 {
		return args[1]
	}
	return NIL
}
func (m *TransientArrayMap) AssocBang(key, val Object) TransientAssociative {
	m.check()
	if i := m.indexOf(key); i >= 0 {
		m.arr[i+1] = val
		return m
	}
	if int64(len(m.arr)) >= HASHMAP_THRESHOLD {
		// The promotion cost is bounded by the small-map threshold. As in
		// Clojure, the returned transient can have a different concrete type.
		h := NewHashMap(m.arr...).AsTransient().(*TransientHashMap)
		return h.AssocBang(key, val)
	}
	m.arr = append(m.arr, key, val)
	return m
}
func (m *TransientArrayMap) WithoutBang(key Object) TransientMapCollection {
	m.check()
	if i := m.indexOf(key); i >= 0 {
		copy(m.arr[i:], m.arr[i+2:])
		m.arr[len(m.arr)-2], m.arr[len(m.arr)-1] = nil, nil
		m.arr = m.arr[:len(m.arr)-2]
	}
	return m
}
func (m *TransientArrayMap) ConjBang(obj Object) TransientCollection {
	m.check()
	return transientMapConj(m, obj)
}
func (m *TransientArrayMap) Persistent() Object {
	m.check()
	m.live = false
	return &ArrayMap{arr: m.arr}
}

func (m *TransientHashMap) Count() int { m.check(); return m.count }
func (m *TransientHashMap) Get(key Object) (bool, Object) {
	m.check()
	if m.root != nil {
		if p := m.root.find(0, key.Hash(), key); p != nil {
			return true, p.Value
		}
	}
	return false, nil
}
func (m *TransientHashMap) Call(args []Object) Object {
	CheckArity(args, 1, 2)
	if ok, val := m.Get(args[0]); ok {
		return val
	}
	if len(args) == 2 {
		return args[1]
	}
	return NIL
}
func (m *TransientHashMap) AssocBang(key, val Object) TransientAssociative {
	m.check()
	added := false
	if m.root == nil {
		m.root = emptyIndexedNode
	}
	m.root = transientNodeAssoc(m.edit, m.root, 0, key.Hash(), key, val, &added)
	if added {
		m.count++
	}
	return m
}
func (m *TransientHashMap) WithoutBang(key Object) TransientMapCollection {
	m.check()
	if m.root != nil {
		removed := false
		m.root = transientNodeWithout(m.edit, m.root, 0, key.Hash(), key, &removed)
		if removed {
			m.count--
		}
	}
	return m
}
func (m *TransientHashMap) ConjBang(obj Object) TransientCollection {
	m.check()
	return transientMapConj(m, obj)
}
func (m *TransientHashMap) Persistent() Object {
	m.check()
	m.live = false
	return &HashMap{count: m.count, root: m.root}
}

func transientMapConj(m TransientMapCollection, obj Object) TransientCollection {
	switch obj := obj.(type) {
	case Vec:
		if obj.Count() != 2 {
			panic(RT.NewError("Vector argument to map's conj must be a vector with two elements"))
		}
		return m.AssocBang(obj.At(0), obj.At(1))
	case Map:
		for iter := obj.Iter(); iter.HasNext(); {
			p := iter.Next()
			m = m.AssocBang(p.Key, p.Value).(TransientMapCollection)
		}
		return m
	default:
		panic(RT.NewError("Argument to map's conj must be a vector with two elements or a map"))
	}
}

func editableBitmap(edit *transientEdit, n *BitmapIndexedNode) *BitmapIndexedNode {
	if n.edit == edit {
		return n
	}
	arr := make([]interface{}, len(n.array))
	copy(arr, n.array)
	return &BitmapIndexedNode{bitmap: n.bitmap, array: arr, edit: edit}
}
func editableArrayNode(edit *transientEdit, n *ArrayNode) *ArrayNode {
	if n.edit == edit {
		return n
	}
	arr := make([]Node, len(n.array))
	copy(arr, n.array)
	return &ArrayNode{count: n.count, array: arr, edit: edit}
}
func editableCollision(edit *transientEdit, n *HashCollisionNode) *HashCollisionNode {
	if n.edit == edit {
		return n
	}
	arr := make([]interface{}, len(n.array))
	copy(arr, n.array)
	return &HashCollisionNode{hash: n.hash, count: n.count, array: arr, edit: edit}
}

func transientNodeAssoc(edit *transientEdit, node Node, shift uint, hash uint32, key, val Object, added *bool) Node {
	switch n := node.(type) {
	case *BitmapIndexedNode:
		bit := bitpos(hash, shift)
		idx := n.index(bit)
		if n.bitmap&bit != 0 {
			k, v := n.array[2*idx], n.array[2*idx+1]
			if k == nil {
				child := transientNodeAssoc(edit, v.(Node), shift+5, hash, key, val, added)
				if child == v {
					return n
				}
				n = editableBitmap(edit, n)
				n.array[2*idx+1] = child
				return n
			}
			if key.Equals(k) {
				n = editableBitmap(edit, n)
				n.array[2*idx+1] = val
				return n
			}
			*added = true
			n = editableBitmap(edit, n)
			n.array[2*idx], n.array[2*idx+1] = nil, createNode(shift+5, k.(Object), v.(Object), hash, key, val)
			return n
		}
		if bitCount(n.bitmap) >= 16 {
			nodes := make([]Node, 32)
			nodes[mask(hash, shift)] = transientNodeAssoc(edit, emptyIndexedNode, shift+5, hash, key, val, added)
			j := 0
			for i := uint(0); i < 32; i++ {
				if n.bitmap&(1<<i) == 0 {
					continue
				}
				if n.array[j] == nil {
					nodes[i] = n.array[j+1].(Node)
				} else {
					unused := false
					k := n.array[j].(Object)
					nodes[i] = transientNodeAssoc(edit, emptyIndexedNode, shift+5, k.Hash(), k, n.array[j+1].(Object), &unused)
				}
				j += 2
			}
			return &ArrayNode{count: bitCount(n.bitmap) + 1, array: nodes, edit: edit}
		}
		n = editableBitmap(edit, n)
		n.array = append(n.array, nil, nil)
		copy(n.array[2*idx+2:], n.array[2*idx:len(n.array)-2])
		n.array[2*idx], n.array[2*idx+1] = key, val
		n.bitmap |= bit
		*added = true
		return n
	case *ArrayNode:
		idx := mask(hash, shift)
		child := n.array[idx]
		if child == nil {
			n = editableArrayNode(edit, n)
			n.array[idx] = transientNodeAssoc(edit, emptyIndexedNode, shift+5, hash, key, val, added)
			n.count++
			return n
		}
		newChild := transientNodeAssoc(edit, child, shift+5, hash, key, val, added)
		if newChild == child {
			return n
		}
		n = editableArrayNode(edit, n)
		n.array[idx] = newChild
		return n
	case *HashCollisionNode:
		if hash == n.hash {
			idx := n.findIndex(key)
			n = editableCollision(edit, n)
			if idx >= 0 {
				n.array[idx+1] = val
				return n
			}
			n.array = append(n.array, key, val)
			n.count++
			*added = true
			return n
		}
		b := &BitmapIndexedNode{bitmap: bitpos(n.hash, shift), array: []interface{}{nil, n}, edit: edit}
		return transientNodeAssoc(edit, b, shift, hash, key, val, added)
	default:
		panic(RT.NewError("Invalid hash map node"))
	}
}

func transientNodeWithout(edit *transientEdit, node Node, shift uint, hash uint32, key Object, removed *bool) Node {
	switch n := node.(type) {
	case *BitmapIndexedNode:
		bit := bitpos(hash, shift)
		if n.bitmap&bit == 0 {
			return n
		}
		idx := n.index(bit)
		k, v := n.array[2*idx], n.array[2*idx+1]
		if k == nil {
			child := transientNodeWithout(edit, v.(Node), shift+5, hash, key, removed)
			if child == v {
				return n
			}
			if child != nil {
				n = editableBitmap(edit, n)
				n.array[2*idx+1] = child
				return n
			}
		} else if key.Equals(k) {
			*removed = true
		} else {
			return n
		}
		if n.bitmap == bit {
			return nil
		}
		n = editableBitmap(edit, n)
		n.bitmap ^= bit
		copy(n.array[2*idx:], n.array[2*idx+2:])
		n.array[len(n.array)-2], n.array[len(n.array)-1] = nil, nil
		n.array = n.array[:len(n.array)-2]
		return n
	case *ArrayNode:
		idx := mask(hash, shift)
		child := n.array[idx]
		if child == nil {
			return n
		}
		newChild := transientNodeWithout(edit, child, shift+5, hash, key, removed)
		if child == newChild {
			return n
		}
		if newChild == nil && n.count <= 8 {
			return n.pack(uint(idx))
		}
		n = editableArrayNode(edit, n)
		n.array[idx] = newChild
		if newChild == nil {
			n.count--
		}
		return n
	case *HashCollisionNode:
		idx := n.findIndex(key)
		if idx < 0 {
			return n
		}
		*removed = true
		if n.count == 1 {
			return nil
		}
		n = editableCollision(edit, n)
		copy(n.array[idx:], n.array[idx+2:])
		n.array[len(n.array)-2], n.array[len(n.array)-1] = nil, nil
		n.array = n.array[:len(n.array)-2]
		n.count--
		return n
	default:
		panic(RT.NewError("Invalid hash map node"))
	}
}
