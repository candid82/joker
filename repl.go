// +build !plan9

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/candid82/joker/core"
	"github.com/peterh/liner"
)

func repl(phase Phase) {
	ProcessReplData()
	GLOBAL_ENV.FindNamespace(MakeSymbol("user")).ReferAll(GLOBAL_ENV.FindNamespace(MakeSymbol("joker.repl")))
	fmt.Printf("Welcome to joker %s. Use EOF (Ctrl-D) or SIGINT (Ctrl-C) to exit.\n", VERSION)
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	replContext := NewReplContext(parseContext.GlobalEnv)

	var runeReader io.RuneReader
	var rl *liner.State
	var historyFilename string
	if noReadline {
		runeReader = bufio.NewReader(Stdin)
	} else {
		historyFilename = filepath.Join(os.TempDir(), ".joker-history")
		rl = liner.NewLiner()
		defer rl.Close()
		rl.SetCtrlCAborts(true)

		if f, err := os.Open(historyFilename); err == nil {
			rl.ReadHistory(f)
			f.Close()
		}

		runeReader = NewLineRuneReader(rl)

		for _, line := range strings.Split(string(dataRead), "\n") {
			if strings.TrimSpace(line) != "" {
				rl.AppendHistory(line)
			}
		}
		dataRead = []rune{}
	}

	reader := NewReader(runeReader, "<repl>")

	for {
		if noReadline {
			print(GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		} else {
			runeReader.(*LineRuneReader).Prompt = (GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "=> ")
		}
		if processReplCommand(reader, phase, parseContext, replContext) {
			if !noReadline {
				if f, err := os.Create(historyFilename); err == nil {
					rl.WriteHistory(f)
					f.Close()
				}
			}
			return
		}
	}
}
