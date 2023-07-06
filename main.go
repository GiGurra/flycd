package main

import (
	"fmt"
	"os"
)

const Version = "v0.0.5"

func main() {
	fmt.Printf("Starting FlyCD %s...\n", Version)

	// print all cli arguments
	for _, arg := range os.Args[1:] {
		print(arg)
	}
}
