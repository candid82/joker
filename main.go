package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	fmt.Println("Welcome to gclojure. Use ctrl-c to exit.")
	s := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		obj, err := Read(s)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("%v\n", obj)
		}
	}
}
