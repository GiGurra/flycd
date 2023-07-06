package main

import "os"

const Version = "v0.0.4"

func main() {
	print("Starting FlyCD...")

	// print all cli arguments
	for _, arg := range os.Args[1:] {
		print(arg)
	}
}
