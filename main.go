package main

import (
	"depudados/commands"
	"log"
)

func main() {
	cmd, err := commands.NewCommand()

	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Execute()

	if err != nil {
		log.Fatal(err)
	}
}
