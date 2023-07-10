package deploy

import (
	"context"
	"flycd/internal/flycd"
	"fmt"
	"github.com/spf13/cobra"
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
			WithForce(*flags.force).
			WithAbortOnFirstError(true)

		ctx := context.Background()

		_, err := flycd.DeployAll(ctx, path, deployCfg)
		if err != nil {
			fmt.Printf("Error deploying: %v\n", err)
			return
		}
	},
}

func init() {
	flags.force = Cmd.Flags().BoolP("force", "f", false, "Force deploy even if no changes detected")
}
