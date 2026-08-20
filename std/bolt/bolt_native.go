package bolt

import (
	"bytes"
	"os"
	"sync/atomic"
	"unsafe"

	. "github.com/candid82/joker/core"
	bolt "go.etcd.io/bbolt"
)

type (
	// TODO: wrapper types like this can probably be auto generated
	BoltDB struct {
		*bolt.DB
		hash uint32
	}
	BoltTx struct {
		state *boltTxState
		hash  uint32
	}
	BoltContext interface {
		Object
		boltContext()
	}
	boltTxState struct {
		tx     *bolt.Tx
		active atomic.Bool
	}
)

var (
	boltContextType *Type
	boltDBType      *Type
	boltTxType      *Type
)

func MakeBoltDB(db *bolt.DB) BoltDB {
	res := BoltDB{db, 0}
	res.hash = HashPtr(uintptr(unsafe.Pointer(db)))
	return res
}

func (db BoltDB) ToString(escape bool) string {
	return "#object[BoltDB]"
}

func (db BoltDB) Equals(other interface{}) bool {
	if otherDb, ok := other.(BoltDB); ok {
		return db.DB == otherDb.DB
	}
	return false
}

func (db BoltDB) GetInfo() *ObjectInfo {
	return nil
}

func (db BoltDB) GetType() *Type {
	return boltDBType
}

func (db BoltDB) Hash() uint32 {
	return db.hash
}

func (db BoltDB) WithInfo(info *ObjectInfo) Object {
	return db
}

func (db BoltDB) boltContext() {}

func MakeBoltTx(tx *bolt.Tx) BoltTx {
	state := &boltTxState{tx: tx}
	state.active.Store(true)
	return BoltTx{
		state: state,
		hash:  HashPtr(uintptr(unsafe.Pointer(state))),
	}
}

func (tx BoltTx) ToString(escape bool) string {
	return "#object[BoltTx]"
}

func (tx BoltTx) Equals(other interface{}) bool {
	if otherTx, ok := other.(BoltTx); ok {
		return tx.state == otherTx.state
	}
	return false
}

func (tx BoltTx) GetInfo() *ObjectInfo {
	return nil
}

func (tx BoltTx) GetType() *Type {
	return boltTxType
}

func (tx BoltTx) Hash() uint32 {
	return tx.hash
}

func (tx BoltTx) WithInfo(info *ObjectInfo) Object {
	return tx
}

func (tx BoltTx) boltContext() {}

func (tx BoltTx) activeTx() *bolt.Tx {
	if tx.state == nil || !tx.state.active.Load() {
		panic(RT.NewError("BoltTx is no longer active"))
	}
	return tx.state.tx
}

func (tx BoltTx) invalidate() {
	if tx.state != nil {
		tx.state.active.Store(false)
	}
}

func EnsureArgIsBoltDB(args []Object, index int) BoltDB {
	obj := args[index]
	if c, yes := obj.(BoltDB); yes {
		return c
	}
	panic(FailArg(obj, "BoltDB", index))
}

func ExtractBoltDB(args []Object, index int) *bolt.DB {
	return EnsureArgIsBoltDB(args, index).DB
}

func EnsureArgIsBoltTx(args []Object, index int) BoltTx {
	obj := args[index]
	if tx, yes := obj.(BoltTx); yes {
		return tx
	}
	panic(FailArg(obj, "BoltTx", index))
}

func ExtractBoltTx(args []Object, index int) *bolt.Tx {
	return EnsureArgIsBoltTx(args, index).activeTx()
}

func ExtractBoltContext(args []Object, index int) BoltContext {
	obj := args[index]
	if context, yes := obj.(BoltContext); yes {
		return context
	}
	panic(FailArg(obj, "BoltDB or BoltTx", index))
}

func open(filename string, mode int) *bolt.DB {
	db, err := bolt.Open(filename, os.FileMode(mode), nil)
	PanicOnErr(err)
	return db
}

func close(db *bolt.DB) Nil {
	err := db.Close()
	PanicOnErr(err)
	return NIL
}

func withView(context BoltContext, fn func(*bolt.Tx)) {
	switch ctx := context.(type) {
	case BoltDB:
		err := ctx.View(func(tx *bolt.Tx) error {
			fn(tx)
			return nil
		})
		PanicOnErr(err)
	case BoltTx:
		fn(ctx.activeTx())
	default:
		panic(RT.NewError("Expected BoltDB or BoltTx"))
	}
}

func withUpdate(context BoltContext, fn func(*bolt.Tx)) {
	switch ctx := context.(type) {
	case BoltDB:
		err := ctx.Update(func(tx *bolt.Tx) error {
			fn(tx)
			return nil
		})
		PanicOnErr(err)
	case BoltTx:
		tx := ctx.activeTx()
		if !tx.Writable() {
			panic(RT.NewError("BoltTx is read-only"))
		}
		fn(tx)
	default:
		panic(RT.NewError("Expected BoltDB or BoltTx"))
	}
}

func runTransaction(db *bolt.DB, fn Callable, writable bool) Object {
	var res Object = NIL
	call := func(tx *bolt.Tx) error {
		wrapped := MakeBoltTx(tx)
		defer wrapped.invalidate()
		res = fn.Call([]Object{wrapped})
		return nil
	}
	var err error
	if writable {
		err = db.Update(call)
	} else {
		err = db.View(call)
	}
	PanicOnErr(err)
	return res
}

func view(db *bolt.DB, fn Callable) Object {
	return runTransaction(db, fn, false)
}

func update(db *bolt.DB, fn Callable) Object {
	return runTransaction(db, fn, true)
}

func createBucket(context BoltContext, name string) Nil {
	withUpdate(context, func(tx *bolt.Tx) {
		_, err := tx.CreateBucket([]byte(name))
		PanicOnErr(err)
	})
	return NIL
}

func createBucketIfNotExists(context BoltContext, name string) Nil {
	withUpdate(context, func(tx *bolt.Tx) {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		PanicOnErr(err)
	})
	return NIL
}

func deleteBucket(context BoltContext, name string) Nil {
	withUpdate(context, func(tx *bolt.Tx) {
		err := tx.DeleteBucket([]byte(name))
		PanicOnErr(err)
	})
	return NIL
}

func getBucket(tx *bolt.Tx, bucket string) *bolt.Bucket {
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		panic(RT.NewError("Bucket doesn't exists: " + bucket))
	}
	return b
}

func nextSequence(context BoltContext, bucket string) int {
	var id uint64
	withUpdate(context, func(tx *bolt.Tx) {
		b := getBucket(tx, bucket)
		var err error
		id, err = b.NextSequence()
		PanicOnErr(err)
	})
	return int(id)
}

func put(context BoltContext, bucket, key, value string) Nil {
	withUpdate(context, func(tx *bolt.Tx) {
		b := getBucket(tx, bucket)
		err := b.Put([]byte(key), []byte(value))
		PanicOnErr(err)
	})
	return NIL
}

func delete(context BoltContext, bucket, key string) Nil {
	withUpdate(context, func(tx *bolt.Tx) {
		b := getBucket(tx, bucket)
		err := b.Delete([]byte(key))
		PanicOnErr(err)
	})
	return NIL
}

func get(context BoltContext, bucket, key string) Object {
	var res Object = NIL
	withView(context, func(tx *bolt.Tx) {
		b := getBucket(tx, bucket)
		if value := b.Get([]byte(key)); value != nil {
			// Bolt values are only valid for the lifetime of the transaction.
			res = MakeString(string(value))
		}
	})
	return res
}

func countByPrefix(context BoltContext, bucket, prefix string) int {
	count := 0
	withView(context, func(tx *bolt.Tx) {
		b := getBucket(tx, bucket)
		c := b.Cursor()
		pr := []byte(prefix)
		for k, _ := c.Seek(pr); k != nil && bytes.HasPrefix(k, pr); k, _ = c.Next() {
			count++
		}
	})
	return count
}

func byPrefix(context BoltContext, bucket, prefix string, opts Map) *ArrayVector {
	limit := -1
	if ok, value := opts.Get(MakeKeyword("limit")); ok {
		limit = EnsureObjectIsInt(value, "limit: %s").I
		if limit < 0 {
			panic(RT.NewError(":limit must be non-negative"))
		}
	}

	res := EmptyArrayVector()
	withView(context, func(tx *bolt.Tx) {
		b := getBucket(tx, bucket)
		if limit == 0 {
			return
		}
		c := b.Cursor()
		pr := []byte(prefix)
		count := 0
		for k, v := c.Seek(pr); k != nil && bytes.HasPrefix(k, pr); k, v = c.Next() {
			res.Append(NewVectorFrom(MakeString(string(k)), MakeString(string(v))))
			count++
			if count == limit {
				break
			}
		}
	})
	return res
}

func init() {
	boltContextType = RegInterface("BoltContext", (*BoltContext)(nil), "Implemented by BoltDB and BoltTx")
	boltDBType = RegType("BoltDB", (*BoltDB)(nil), "Wraps Bolt DB type")
	boltTxType = RegType("BoltTx", (*BoltTx)(nil), "Wraps a transaction-scoped Bolt Tx type")
}
