package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/candid82/joker/core"
)

func TestReplUsesVMAndRecovers(t *testing.T) {
	core.ProcessCoreData()
	var out, errors bytes.Buffer
	oldOut, oldErr := core.Stdout, core.Stderr
	core.Stdout, core.Stderr = &out, &errors
	defer func() { core.Stdout, core.Stderr = oldOut, oldErr }()
	ctx := &core.ParseContext{GlobalEnv: core.GLOBAL_ENV}
	repl := NewReplContext(core.GLOBAL_ENV)
	reader := core.NewReader(strings.NewReader("(meta ^{:a 1} [])\n(nth [] 3)\n(+ 1 2)\n"), "<repl-test>")
	for !processReplCommand(reader, core.EVAL, ctx, repl) {
	}
	if out.String() != "{:a 1}\n3\n" {
		t.Fatalf("REPL results: %q", out.String())
	}
	if strings.Contains(errors.String(), "Runtime AST execution") || !strings.Contains(errors.String(), "out of bounds") {
		t.Fatalf("REPL error: %s", errors.String())
	}
	if !repl.first.Value.Equals(core.Int{I: 3}) {
		t.Fatal("REPL value context not updated")
	}
}
