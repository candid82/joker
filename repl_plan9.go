package main

import (
	"bufio"
	"fmt"
	"io"

	. "github.com/candid82/joker/core"
)

func repl(phase Phase) {
	ProcessReplData()
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.FindNamespace(MakeSymbol("joker.repl")))
	fmt.Printf("Welcome to joker %s. Use '(exit)', %s to exit.\n", VERSION, EXITERS)
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	replContext := NewReplContext(parseContext.GlobalEnv)

	var runeReader io.RuneReader
	runeReader = bufio.NewReader(Stdin)
	reader := NewReader(runeReader, "<repl>")

	for {
		print(GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		if processReplCommand(reader, phase, parseContext, replContext) {
			return
		}
	}
}
