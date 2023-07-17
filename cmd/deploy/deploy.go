package deploy

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/pkg/domain"
	"github.com/gigurra/flycd/pkg/domain/model"
	"github.com/gigurra/flycd/pkg/util/util_cobra"
	"github.com/spf13/cobra"
	"os"
)

type flags struct {
	force      *bool
	abortEarly *bool
}

func (f *flags) Init(cmd *cobra.Command) {
	f.force = cmd.Flags().BoolP("force", "f", false, "Force deploy even if no changes detected")
	f.abortEarly = cmd.Flags().BoolP("abort-early", "a", false, "Abort on first error")
}

func Cmd(deployService domain.DeployService) *cobra.Command {
	flags := flags{}
	return util_cobra.CreateCmd(&flags, func() *cobra.Command {
		return &cobra.Command{
			Use:   "deploy <path>",
			Short: "Manually deploy a single domain app, or all domain apps inside a folder",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				path := args[0]
				fmt.Printf("Deploying from: %s\n", path)

				deployCfg := model.
					NewDefaultDeployConfig().
					WithRetries(1).
					WithForce(*flags.force).
					WithAbortOnFirstError(*flags.abortEarly)

				ctx := context.Background()

				result, err := deployService.DeployAll(ctx, path, deployCfg)
				if err != nil {
					fmt.Printf("Error deploying: %v\n", err)
					return
				}

				fmt.Printf("Deployed %d projects\n", len(result.ProcessedProjects))
				for _, success := range result.ProcessedProjects {
					fmt.Printf(" - %s\n", success.ProjectConfig.Project)
				}

				fmt.Printf("Deployed %d apps\n", len(result.SucceededApps))
				for _, success := range result.SucceededApps {
					fmt.Printf(" - %s (%s)\n", success.Spec.AppConfig.App, success.SuccessType)
				}

				if !result.Success() {
					fmt.Printf("Failed to deploy %d apps\n", len(result.FailedApps))
					for _, failure := range result.FailedApps {
						fmt.Printf(" - %s: %v\n", failure.Spec.AppConfig.App, failure.Cause)
					}

					fmt.Printf("Failed to deploy %d projects\n", len(result.FailedProjects))
					for _, failure := range result.FailedProjects {
						fmt.Printf(" - %s: %v\n", failure.Spec.ProjectConfig.Project, failure.Cause)
					}
					os.Exit(1)
				}
			},
		}
	})
}
