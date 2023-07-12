package deploy

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd"
	"github.com/spf13/cobra"
	"os"
)

var flags struct {
	force *bool
}

var Cmd = &cobra.Command{
	Use:   "deploy <path>",
	Short: "Manually deploy a single flycd app, or all flycd apps inside a folder",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		fmt.Printf("Deploying from: %s\n", path)

		deployCfg := flycd.
			NewDeployConfig().
			WithRetries(1).
			WithForce(*flags.force).
			WithAbortOnFirstError(true)

		ctx := context.Background()

		result, err := flycd.DeployAll(ctx, path, deployCfg)
		if err != nil {
			fmt.Printf("Error deploying: %v\n", err)
			return
		}

		fmt.Printf("Deployed %d apps\n", len(result.SucceededApps))
		for _, success := range result.SucceededApps {
			fmt.Printf(" - %s @ %s\n", success.AppConfig.App, success.Path)
		}

		if !result.Success() {
			fmt.Printf("Failed to deploy %d apps\n", len(result.FailedApps))
			for _, failure := range result.FailedApps {
				fmt.Printf(" - %s @ %s: %v\n", failure.Spec.AppConfig.App, failure.Spec.Path, failure.Cause)
			}
			os.Exit(1)
		}
	},
}

func init() {
	flags.force = Cmd.Flags().BoolP("force", "f", false, "Force deploy even if no changes detected")
}
