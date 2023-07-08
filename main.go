package main

import (
	"flycd/internal/flycd"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

const Version = "v0.0.5"

var rootCmd = &cobra.Command{
	Use:   "flycd",
	Short: "flycd deployment of fly apps entirely from code, without manual flyctl commands... I hope :D",
	Long:  `Complete documentation is available at https://github.com/gigurra/flycd`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		fmt.Printf("Hello World, args: %+v\n", args)
		if len(args) == 1 {
			fmt.Printf("Deploying from: %s\n", args[0])
			err := flycd.Deploy(args[0])
			if err != nil {
				fmt.Println("Error deploying:", err)
				os.Exit(1)
			}
		} else if len(args) == 0 {
			// use current workdir
			cwd := os.Getenv("PWD")
			fmt.Printf("Using current workdir: %s\n", cwd)
			err := flycd.Deploy(cwd)
			if err != nil {
				fmt.Println("Error deploying:", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Usage: flycd [path]\n")
			os.Exit(1)
		}
	},
}

func main() {
	fmt.Printf("Starting FlyCD %s...\n", Version)

	// Check that required applications are installed
	requiredApps := []string{"flyctl", "git", "yj"}
	for _, app := range requiredApps {
		_, err := exec.LookPath(app)
		if err != nil {
			fmt.Printf("Error: required app '%s' not found in PATH\n", app)
			os.Exit(1)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// print all cli arguments
	for _, arg := range os.Args[1:] {
		print(arg)
	}
}
