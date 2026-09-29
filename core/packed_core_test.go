//go:build packed_core
// +build packed_core

package core

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	GLOBAL_ENV.InitEnv(Stdin, Stdout, Stderr, nil)
	RT.GIL.Lock()
	ProcessCoreData()
	GLOBAL_ENV.ReferCoreToUser()
	RT.GIL.Unlock()
	os.Exit(m.Run())
}
