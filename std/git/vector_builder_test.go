package git

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	. "github.com/candid82/joker/core"
)

func TestCommitParentVectorBuilder(t *testing.T) {
	for _, n := range []int{0, 2, 33} {
		parents := make([]plumbing.Hash, n)
		for i := range parents {
			parents[i][0] = byte(i + 1)
		}
		_, value := makeCommit(&object.Commit{ParentHashes: parents}).Get(MakeKeyword("parent-hashes"))
		v := value.(*Vector)
		if v.Count() != n {
			t.Fatal("incorrect parent hash count")
		}
		for i, hash := range parents {
			if !v.At(i).Equals(MakeString(hash.String())) {
				t.Fatal("parent hash values or order changed")
			}
		}
	}
}
