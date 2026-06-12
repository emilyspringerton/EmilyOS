package main

import (
	"flag"
	"fmt"
	"os"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("emilyos %s\n", Version)
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "emilyos %s — policy kernel\n", Version)
	fmt.Fprintf(os.Stderr, "Usage: emilyos [--version]\n")
	os.Exit(1)
}
