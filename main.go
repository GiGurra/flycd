package main

import (
	"flycd/internal/flycd"
	"flycd/internal/flyctl"
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
		if len(args) < 1 {
			err := cmd.Help() // Display help message
			if err != nil {
				fmt.Printf("Error displaying help: %v\n", err)
			}
			err = cmd.Usage()
			if err != nil {
				fmt.Printf("error displaying usage: %v\n", err)
			}
			os.Exit(1) // Exit with code 1
		}
	},
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Manually deploy a single flycd app, or all flycd apps inside a folder",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		fmt.Printf("Deploying from: %s\n", path)

		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			fmt.Printf("Error getting force flag: %v\n", err)
			os.Exit(1)
		}

		err = flycd.Deploy(path, force)
		if err != nil {
			fmt.Printf("Error deploying from %s: %v\n:", path, err)
			os.Exit(1)
		}
	},
}

var _ any = deployCmd.Flags().BoolP("force", "f", false, "Force re-deploy even if there are no changes")

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor a single flycd app, or all flycd apps inside a folder (recursively)",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		path, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}

		if len(args) > 0 {
			path = args[0]
		}

		fmt.Printf("Monitoring: %s\n", path)

		fmt.Printf("Not implemented yet, sorry :(\n")
		os.Exit(1)
	},
}

func OrgSlugArg() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("accepts %d arg(s) = fly.io org slug, received %d. Use 'flyctl orgs list' to find yours", 1, len(args))
		}
		return nil
	}
}

var installCmd = &cobra.Command{
	Use:   "install <fly.io org slug>",
	Short: "Install flycd into your fly.io account, listening to webhooks from this cfg repo and your app repos",
	Args:  OrgSlugArg(),
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Printf("Installing flycd to your fly.io account. Checking what orgs you have access to...\n")
		orgSlug := args[0]

		fmt.Printf("Step 1 is to create an organisation token with which flycd can access your fly.io account\n")
		token, err := flyctl.CreateOrgToken(orgSlug)
		if err != nil {
			fmt.Printf("Error creating org token: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Token created: %s\n", token)

		fmt.Printf("Not implemented yet, sorry :(\n")
		os.Exit(1)
	},
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade your flycd installation in your fly.io account to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		path, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}

		if len(args) > 0 {
			path = args[0]
		}

		fmt.Printf("Monitoring: %s\n", path)

		fmt.Printf("Not implemented yet, sorry :(\n")
		os.Exit(1)
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall flycd from your fly.io account",
	Run: func(cmd *cobra.Command, args []string) {
		path, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}

		if len(args) > 0 {
			path = args[0]
		}

		fmt.Printf("Monitoring: %s\n", path)

		fmt.Printf("Not implemented yet, sorry :(\n")
		os.Exit(1)
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
	rootCmd.AddCommand(deployCmd, monitorCmd, installCmd, upgradeCmd, uninstallCmd)

	// run cli
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("FlyCD %s exiting normally, bye!\n", Version)
}
