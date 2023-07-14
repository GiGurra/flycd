package main

import (
	_ "embed"
	"fmt"
	"github.com/gigurra/flycd/cmd/convert"
	"github.com/gigurra/flycd/cmd/deploy"
	"github.com/gigurra/flycd/cmd/install"
	"github.com/gigurra/flycd/cmd/monitor"
	"github.com/gigurra/flycd/cmd/repos"
	"github.com/gigurra/flycd/internal/flycd"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

const Version = "v0.0.29"

var rootCmd = &cobra.Command{
	Use:   "flycd",
	Short: "flycd deployment of fly apps entirely from code, without manual fly.io cli commands... I hope :D",
	Long:  `Complete documentation is available at https://github.com/gigurra/flycd`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			err := cmd.Usage()
			if err != nil {
				fmt.Printf("error displaying usage: %v\n", err)
			}
			os.Exit(1) // Exit with code 1
		}
	},
}

func main() {
	fmt.Printf("Starting FlyCD %s...\n", Version)

	// Check that required applications are installed
	// At some point we should just use a go library instead,
	// and maybe even embed fly.io cli into our app :P
	// (Alternatively we could integrate directly with fly.io API)
	requiredApps := []string{"fly", "git", "ssh"}
	for _, app := range requiredApps {
		_, err := exec.LookPath(app)
		if err != nil {
			fmt.Printf("Error: required app '%s' not found in PATH\n", app)
			os.Exit(1)
		}
	}

	// Create services (sigh, feels like java again :S)
	deployService := flycd.NewDeployService()
	webhookService := flycd.NewWebHookService(deployService)

	// prepare cli
	rootCmd.AddCommand(
		deploy.Cmd(deployService),
		monitor.Cmd(deployService, webhookService),
		install.Cmd(PackagedFileSystem, deployService),
		convert.Cmd,
		repos.Cmd,
	)

	// run cli
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("FlyCD %s exiting normally, bye!\n", Version)
}
