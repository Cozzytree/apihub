package main

import (
	"os"

	"github.com/Cozzytree/apihub/cli"
)

func main() {
	args := os.Args
	err := cli.Init().Run(args)
	if err != nil {
		os.Exit(1)
	}
}
