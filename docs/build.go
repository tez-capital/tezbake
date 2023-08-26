package main

import (
	"log"

	"alis.is/bb-cli/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	err := doc.GenMarkdownTree(cmd.RootCmd, "./bin/docs")
	if err != nil {
		log.Fatal(err)
	}
}
