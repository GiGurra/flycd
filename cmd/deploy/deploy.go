package deploy

import (
	"context"
	"flycd/internal/flycd"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var Cmd = &cobra.Command{
	Use:   "deploy",
	Short: "Manually deploy a single flycd app, or all flycd apps inside a folder",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		fmt.Printf("Deploying from: %s\n", path)

		ctx := context.Background()
		err := flycd.Deploy(ctx, path, *forceFlag)
		if err != nil {
			fmt.Printf("Error deploying from %s: %v\n:", path, err)
			os.Exit(1)
		}
	},
}

var forceFlag *bool

func init() {
	forceFlag = Cmd.Flags().BoolP("force", "f", false, "Force re-deploy even if there are no changes")
}
