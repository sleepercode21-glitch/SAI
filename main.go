package main

import (
	"fmt"
	"os"

	"github.com/sleepercode/sai/cli"
)

func main() {
	app := cli.NewApp()
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "sai: %v\n", err)
		os.Exit(1)
	}
}
