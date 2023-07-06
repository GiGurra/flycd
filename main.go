package main

import "os"

const version = "v0.0.1"

func main() {
	print("Starting FlyCD...")

	// print all cli arguments
	for _, arg := range os.Args[1:] {
		print(arg)
	}
}
