package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func readFile(filename string) {
	var reader *Reader
	if filename == "--" {
		reader = NewReader(bufio.NewReader(os.Stdin))
	} else {
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: ", err)
			return
		}
		reader = NewReader(bufio.NewReader(f))
	}
	for {
		_, err := TryRead(reader)
		switch {
		case err == io.EOF:
			return
		case err != nil:
			fmt.Fprintln(os.Stderr, "Error: ", err)
			return
		}
	}
}

func skipRestOfLine(reader *Reader) {
	for {
		switch reader.Get() {
		case EOF, '\n':
			return
		}
	}
}

func repl() {
	fmt.Println("Welcome to gclojure. Use ctrl-c to exit.")
	reader := NewReader(bufio.NewReader(os.Stdin))
	for {
		fmt.Print("> ")
		obj, err := TryRead(reader)
		switch {
		case err == io.EOF:
			return
		case err != nil:
			fmt.Fprintln(os.Stderr, "Read error: ", err)
			skipRestOfLine(reader)
		default:
			expr, err := parse(obj)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Parse error: ", err)
				continue
			}
			res, err := expr.Eval()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Eval error: ", err)
				continue
			}
			fmt.Println(res.ToString(true))
		}
	}
}

func main() {
	if len(os.Args) > 1 {
		readFile(os.Args[1])
	} else {
		repl()
	}
}
