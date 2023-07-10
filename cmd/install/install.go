package install

import (
	"context"
	"flycd/internal/flycd"
	"flycd/internal/flyctl"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var Cmd = &cobra.Command{
	Use:   "install <flycd app name> <fly.io org slug> <fly.io region>",
	Short: "Install flycd into your fly.io account, listening to webhooks from this cfg repo and your app repos",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {

		appName := args[0]

		orgSlug := args[1]

		region := args[2]

		fmt.Printf("Installing flycd with app name '%s' to org '%s'\n", appName, orgSlug)

		ctx := context.Background()

		fmt.Printf("Check if flycd app already exists\n")
		appExists, err := flycd.AppExists(ctx, appName)
		if err != nil {
			fmt.Printf("Error checking if app exists: %v\n", err)
			os.Exit(1)
		}

		if !appExists {

			fmt.Printf("Creating a dummy app '%s' to reserve the name\n", appName)
			err = flycd.DeployAppFromConfig(ctx, false, flycd.AppConfig{
				App:           appName,
				Org:           orgSlug,
				PrimaryRegion: region,
				Source:        flycd.NewInlineDockerFileSource("FROM nginx:latest"),
				LaunchParams:  flycd.NewDefaultLaunchParams(appName, orgSlug),
				Services:      []flycd.Service{flycd.NewDefaultServiceConfig()},
			})
			if err != nil {
				fmt.Printf("Error creating dummy app: %v\n", err)
				os.Exit(1)
			}
		}

		existsAccessTokenSecret, err := flyctl.ExistsSecret(ctx, flyctl.ExistsSecretCmd{
			AppName:    appName,
			SecretName: "FLY_ACCESS_TOKEN",
		})
		if err != nil {
			fmt.Printf("Error checking if access token secret exists: %v\n", err)
			os.Exit(1)
		}

		if !existsAccessTokenSecret {

			fmt.Printf("App name successfully reserved... creating access token for org '%s'\n", orgSlug)
			token, err := flyctl.CreateOrgToken(orgSlug)
			if err != nil {
				fmt.Printf("Error creating org token: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Token created.. storing it...\n")

			err = flyctl.StoreSecret(ctx, flyctl.StoreSecretCmd{
				AppName:     appName,
				SecretName:  "FLY_ACCESS_TOKEN",
				SecretValue: token,
			})

			if err != nil {
				fmt.Printf("Error storing token: %v\n", err)
				os.Exit(1)
			}

		}

		wd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Deploying flycd in monitoring mode to fly.io\n")
		err = flycd.DeployAppFromConfig(ctx, false, flycd.AppConfig{
			App:           appName,
			Org:           orgSlug,
			PrimaryRegion: region,
			Source:        flycd.NewLocalFolderSource(wd),
			LaunchParams:  flycd.NewDefaultLaunchParams(appName, orgSlug),
			DeployParams:  flycd.NewDefaultDeployParams(),
			Services:      []flycd.Service{flycd.NewDefaultServiceConfig()},
		})
		if err != nil {
			fmt.Printf("Error deploying flycd in monitoring mode: %v\n", err)
			os.Exit(1)
		}
		// TODO: Add ssh keys as secrets so we can pull from other git repos argocd style
	},
}
