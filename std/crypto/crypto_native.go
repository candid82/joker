package crypto

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	. "github.com/candid82/joker/core"
)

func hmacSum(algorithm, message, key string) string {
	var h func() hash.Hash
	switch algorithm {
	case ":sha1":
		h = sha1.New
	case ":sha224":
		h = sha256.New224
	case ":sha256":
		h = sha256.New
	case ":sha384":
		h = sha512.New384
	case ":sha512":
		h = sha512.New
	default:
		panic(RT.NewError("Unsupported algorithm " + algorithm +
			". Supported algorithms are: :sha1, :sha224, :sha256, :sha384, :sha512"))
	}
	mac := hmac.New(h, []byte(key))
	mac.Write([]byte(message))
	return string(mac.Sum(nil))
}
