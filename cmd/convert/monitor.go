package monitor

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var flags struct {
}

var Cmd = &cobra.Command{
	Use:   "convert <path>",
	Short: "Convert app/apps from fly.toml(s) to/from app.yaml(s)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		argPath := args[0]
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}

		path := func() string {
			if filepath.IsAbs(argPath) {
				return argPath
			} else {
				return filepath.Join(cwd, argPath)
			}
		}()

		fmt.Printf("Preparing files inside: %s\n", path)

		//ctx := context.Background()

		println("Not implemented yet")

	},
}

func init() {
}
