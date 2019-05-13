package filepath

import (
	"os"
	"path/filepath"

	. "github.com/candid82/joker/core"
)

func fileSeq(root string) *Vector {
	res := EmptyVector()
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		PanicOnErr(err)
		m := FileInfoMap(path, info)
		res = res.Conjoin(m)
		return nil
	})
	return res
}

func glob(pattern string) *Vector {
	res := EmptyVector()
	matches, err := filepath.Glob(pattern)
	PanicOnErr(err)
	for _, match := range matches {
		res = res.Conjoin(MakeString(match))
	}
	return res
}
