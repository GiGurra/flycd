package main

import (
	"flycd/internal/flycd"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

const Version = "v0.0.5"

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Manually deploy a single flycd app, or all flycd apps inside a folder",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("deploy called")
		path := args[0]
		fmt.Printf("Deploying from: %s\n", path)
		err := flycd.Deploy(path)
		if err != nil {
			fmt.Printf("Error deploying from %s: %v\n:", path, err)
			os.Exit(1)
		}
	},
}

var rootCmd = &cobra.Command{
	Use:   "flycd",
	Short: "flycd deployment of fly apps entirely from code, without manual flyctl commands... I hope :D",
	Long:  `Complete documentation is available at https://github.com/gigurra/flycd`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			err := cmd.Help() // Display help message
			if err != nil {
				fmt.Printf("Error displaying help: %v\n", err)
			}
			os.Exit(1) // Exit with code 1
		}
	},
}

func main() {
	fmt.Printf("Starting FlyCD %s...\n", Version)

	// Check that required applications are installed
	requiredApps := []string{"flyctl", "git", "ssh", "yj", "cat", "cp"}
	for _, app := range requiredApps {
		_, err := exec.LookPath(app)
		if err != nil {
			fmt.Printf("Error: required app '%s' not found in PATH\n", app)
			os.Exit(1)
		}
	}

	// prepare cli
	rootCmd.AddCommand(deployCmd)

	// run cli
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("FlyCD %s exiting normally, bye!\n", Version)
}
