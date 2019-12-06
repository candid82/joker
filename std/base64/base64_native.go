package base64

import (
	"encoding/base64"

	. "github.com/candid82/joker/core"
)

func decodeString(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(RT.NewError("Invalid base64 string: " + err.Error()))
	}
	return string(decoded)
}

func encodeString(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
