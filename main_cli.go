//go:build !js

package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var output string
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&output, "o", "", "output file path")
	if err := flags.Parse(os.Args[1:]); err != nil {
		exitErr(err)
	}
	inputPath := ""
	if flags.NArg() > 0 {
		inputPath = flags.Arg(0)
	}

	data, err := readInput(inputPath)
	if err != nil {
		exitErr(err)
	}
	decompiled, err := decompileData(data)
	if err != nil {
		exitErr(err)
	}
	if output == "" && inputPath != "" {
		output = defaultOutputPath(inputPath)
	}
	if output != "" {
		if err := os.WriteFile(output, []byte(decompiled), 0644); err != nil {
			exitErr(err)
		}
		fmt.Fprintln(os.Stderr, "wrote", output)
		return
	}
	fmt.Print(decompiled)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
