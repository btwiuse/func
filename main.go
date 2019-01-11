package main

import (
	"fmt"
	"os"

	cmd "github.com/func/func/cmd/func"
)

func main() {
	err := cmd.Func.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
