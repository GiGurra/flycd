package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var flags struct {
	force *bool
}

var Cmd = &cobra.Command{
	Use:   "repos <path>",
	Short: "Traverse the project structure and list all git repos referenced. Useful for finding your dependencies (and setting up webhooks).",
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

		fmt.Printf("Scanning for git repos referenced inside project @ %s\n", path)

		appRepos := make([]model.AppNode, 0)
		projectRepos := make([]model.ProjectNode, 0)

		ctx := context.Background()
		err = flycd.TraverseDeepAppTree(path, model.TraverseAppTreeContext{
			Context: ctx,
			ValidAppCb: func(node model.AppNode) error {
				fmt.Printf("Checking app %s @ %s...\n", node.AppConfig.App, node.Path)
				if node.AppConfig.Source.Repo != "" {
					appRepos = append(appRepos, node)
				}
				return nil
			},
			BeginProjectCb: func(node model.ProjectNode) error {
				if node.IsValidProject() {
					fmt.Printf("Checking project %s @ %s...\n", node.ProjectConfig.Project, node.Path)
					if node.ProjectConfig.Source.Repo != "" {
						projectRepos = append(projectRepos, node)
					}
				} else {
					fmt.Printf("Skipping project (invalid) %s @ %s\n", node.ProjectConfig.Project, node.Path)
				}
				return nil
			},
		})

		fmt.Printf("Found the following project repo references:\n")
		for _, node := range projectRepos {
			srcJson, err := json.Marshal(node.ProjectConfig.Source)
			if err != nil {
				fmt.Printf("Error marshalling source config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf(" - %s @ %s\n", node.ProjectConfig.Project, srcJson)
		}

		fmt.Printf("Found the following app repo references:\n")
		for _, node := range appRepos {
			srcJson, err := json.Marshal(node.AppConfig.Source)
			if err != nil {
				fmt.Printf("Error marshalling source config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf(" - %s @ %s\n", node.AppConfig.App, srcJson)
		}

		if err != nil {
			fmt.Printf("Error walking path %s: %v\n", path, err)
			os.Exit(1)
		}

	},
}

func init() {
	flags.force = Cmd.Flags().BoolP("force", "f", false, "Force overwrite of existing files")
}
