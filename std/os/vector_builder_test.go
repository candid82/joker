package os

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/candid82/joker/core"
)

func TestReadDirVectorBuilder(t *testing.T) {
	dir := t.TempDir()
	if got := readDir(dir).(*Vector); got.Count() != 0 {
		t.Fatal("empty directory must return an empty Vector")
	}
	for _, name := range []string{"b", "a"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got := readDir(dir).(*Vector)
	if got.Count() != 2 {
		t.Fatal("incorrect directory entry count")
	}
	for i, name := range []string{"a", "b"} {
		if ok, value := got.At(i).(Map).Get(MakeKeyword("name")); !ok || !value.Equals(MakeString(name)) {
			t.Fatal("directory entry order or name changed")
		}
	}
}
