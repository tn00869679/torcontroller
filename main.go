package main

import (
	"fmt"
	"os"

	"github.com/Seicrypto/torcontroller/cmd"
)

func main() {
	rootCommand := cmd.InitCommands()
	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
