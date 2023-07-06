package main

import "os"

func main() {
	print("Starting FlyCD...")

	// print all cli arguments
	for _, arg := range os.Args[1:] {
		print(arg)
	}
}
