package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gigurra/flycd/pkg/domain"
	"github.com/gigurra/flycd/pkg/domain/model"
	"github.com/gigurra/flycd/pkg/util/util_cobra"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

type flags struct {
	force *bool
}

func (f *flags) Init(cmd *cobra.Command) {
	f.force = cmd.Flags().BoolP("force", "f", false, "Force overwrite of existing files")
}

func Cmd(ctx context.Context) *cobra.Command {
	flags := flags{}
	return util_cobra.CreateCmd(&flags, func() *cobra.Command {
		return &cobra.Command{
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

				appRepos := make([]model.AppAtFsNode, 0)
				projectRepos := make([]model.ProjectAtFsNode, 0)

				err = domain.TraverseDeepAppTree(path, model.TraverseAppTreeContext{
					Context: ctx,
					ValidAppCb: func(ctx model.TraverseAppTreeContext, node model.AppAtFsNode) error {
						fmt.Printf("Checking app %s @ %s...\n", node.AppConfig.App, node.Path)
						if node.AppConfig.Source.Repo != "" {
							appRepos = append(appRepos, node)
						}
						return nil
					},
					BeginProjectCb: func(ctx model.TraverseAppTreeContext, node model.ProjectAtFsNode) error {
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

	})
}
