package main

import (
	"fmt"
	"os"
	fs "simplefs/internal/shell"
	"strconv"
)

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Usage: simplefs <path_to_data_file> <number_of_blocks>")
		return
	}

	dataPath := os.Args[1]
	numberOfBlocks := os.Args[2]

	numberOfBlocksInt, err := strconv.Atoi(numberOfBlocks)
	if err != nil {
		fmt.Println("error: invalid number_of_blocks value (use a valid number)")
		os.Exit(1)
	}
	shell := fs.NewShell(dataPath, numberOfBlocksInt)
	shell.Init()

}
