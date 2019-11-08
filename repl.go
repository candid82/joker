// +build !plan9

package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	. "github.com/candid82/joker/core"
	"github.com/chzyer/readline"
)

func repl(phase Phase) {
	ProcessReplData()
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.FindNamespace(MakeSymbol("joker.repl")))
	fmt.Printf("Welcome to joker %s. Use EOF (Ctrl-D) or SIGINT (Ctrl-C) to exit.\n", VERSION)
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	replContext := NewReplContext(parseContext.GlobalEnv)

	var runeReader io.RuneReader
	var rl *readline.Instance
	var err error
	if noReadline {
		runeReader = bufio.NewReader(Stdin)
	} else {
		rl, err = readline.New("")
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}
		defer rl.Close()
		runeReader = NewLineRuneReader(rl)
		for _, line := range strings.Split(string(dataRead), "\n") {
			rl.SaveHistory(line)
		}
		dataRead = []rune{}
	}

	reader := NewReader(runeReader, "<repl>")

	for {
		if noReadline {
			print(GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		} else {
			rl.SetPrompt(GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		}
		if processReplCommand(reader, phase, parseContext, replContext) {
			return
		}
	}
}
