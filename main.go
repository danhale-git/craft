package main

import (
	"fmt"
	"os"

	"github.com/danhale-git/craft/cmd"
)

func main() {
	rootCmd := cmd.InitCobra()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
