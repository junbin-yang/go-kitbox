package main

import (
	"fmt"
	"os"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "gen":
		runGen(os.Args[2:])
	case "docs":
		runDocs(os.Args[2:])
	case "debug":
		runDebug(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("binpack version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("binpack - Binary protocol toolkit")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  binpack <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  gen      Generate static encoder/decoder code")
	fmt.Println("  docs     Generate protocol documentation")
	fmt.Println("  debug    Debug binary data with protocol definition")
	fmt.Println("  version  Show version information")
	fmt.Println("  help     Show this help message")
	fmt.Println()
	fmt.Println("Run 'binpack <command> -h' for more information on a command.")
}
