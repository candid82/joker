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
			fmt.Fprintln(os.Stderr, "Error: ", err)
			skipRestOfLine(reader)
		default:
			fmt.Println(obj.ToString(true))
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
