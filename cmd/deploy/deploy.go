package deploy

import (
	"context"
	"flycd/internal/flycd"
	"fmt"
	"github.com/samber/lo"
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

		analysis, err := flycd.AnalyseSpec(path)
		if err != nil {
			fmt.Printf("Error analysing %s: %v\n:", path, err)
			os.Exit(1)
		}

		nodeList, err := analysis.Flatten()
		if err != nil {
			fmt.Printf("Error flattening analysis of %s: %v\n:", path, err)
			os.Exit(1)
		}

		apps := lo.Filter(nodeList, func(node flycd.SpecNode, _ int) bool {
			return node.IsAppNode()
		})

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
		/*
			err := flycd.Deploy(ctx, path, deployCfg)
			if err != nil {
				fmt.Printf("Error deploying from %s: %v\n:", path, err)
				os.Exit(1)
			}*/
	},
}

func init() {
	flags.force = Cmd.Flags().BoolP("force", "f", false, "Force deploy even if no changes detected")
}
