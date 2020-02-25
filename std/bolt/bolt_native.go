package bolt

import (
	. "github.com/candid82/joker/core"
	bolt "go.etcd.io/bbolt"
	"os"
	"unsafe"
)

type (
	// TODO: wrapper types like this can probably be auto generated
	BoltDB struct {
		*bolt.DB
		hash uint32
	}
)

var boltDBType *Type

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

func EnsureBoltDB(args []Object, index int) BoltDB {
	switch c := args[index].(type) {
	case BoltDB:
		return c
	default:
		panic(RT.NewArgTypeError(index, c, "BoltDB"))
	}
}

func ExtractBoltDB(args []Object, index int) *bolt.DB {
	return EnsureBoltDB(args, index).DB
}

func open(filename string, mode int, options Map) *bolt.DB {
	// TODO: handle options
	db, err := bolt.Open(filename, os.FileMode(mode), nil)
	PanicOnErr(err)
	return db
}

func close(db *bolt.DB) Nil {
	PanicOnErr(db.Close())
	return NIL
}

// func update(db *bolt.DB, f Callable) Nil {
// 	db.Update(func(tx *bolt.Tx) error {
// 		response := f.Call([]Object{})

// 		return nil
// 	})
// 	return NIL
// }

func createBucket(db *bolt.DB, name string) Nil {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(name))
		PanicOnErr(err)
		return nil
	})
	return NIL
}

func put(db *bolt.DB, bucket, key, value string) Nil {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			panic(RT.NewError("Bucket doesn't exists: " + bucket))
		}
		err := b.Put([]byte(key), []byte(value))
		PanicOnErr(err)
		return nil
	})
	return NIL
}

func get(db *bolt.DB, bucket, key string) Object {
	var v []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			panic(RT.NewError("Bucket doesn't exists: " + bucket))
		}
		v = b.Get([]byte(key))
		return nil
	})
	if v == nil {
		return NIL
	}
	return MakeString(string(v))
}

func init() {
	boltDBType = RegType("BoltDB", (*BoltDB)(nil), "Wraps Bolt DB type")
}
