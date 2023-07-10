package deploy

import (
	"context"
	"flycd/internal/flycd"
	"fmt"
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
			WithRetries(0).
			WithForce(*flags.force)

		ctx := context.Background()

		apps, err := flycd.ScanForApps(path)
		if err != nil {
			fmt.Printf("Error finding apps: %v\n", err)
			os.Exit(1)
		}

		for _, app := range apps {
			fmt.Printf("Considering app %s @ %s\n", app.AppConfig.App, app.Path)
			if app.IsValidApp() {
				err := flycd.Deploy(ctx, app.Path, deployCfg)
				if err != nil {
					fmt.Printf("Error deploying %s @ %s: %v\n:", app.AppConfig.App, app.Path, err)
					os.Exit(1)
				}
			} else {
				fmt.Printf("App is NOT valid, skipping!\n")
			}
		}
	},
}

func init() {
	flags.force = Cmd.Flags().BoolP("force", "f", false, "Force deploy even if no changes detected")
}
